package requester

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Collector handles sending HTTP requests and collecting statistics
type Collector struct {
	config     Config
	httpClient *http.Client

	// Statistics
	totalRequests atomic.Int64
	successful    atomic.Int64
	failed        atomic.Int64

	// Per-worker response time collection (lock-free during collection)
	workerResponseTimes [][]float64
	workerSamples       [][]ResponseTimeSnapshot
	maxSamples          int
}

// NewCollector creates a new request collector
func NewCollector(config Config) *Collector {
	numWorkers := 16

	// Configure HTTP transport for short connections
	// Client-side close: client actively closes connections after each request
	// Combined with tcp_tw_reuse to avoid port exhaustion
	transport := &http.Transport{
		MaxIdleConns:        0,
		MaxIdleConnsPerHost: 0,
		DisableKeepAlives:   true, // Client closes connection after each request
		// Note: tcp_tw_reuse must be enabled on client to reuse TIME_WAIT ports
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	// Pre-allocate per-worker slices to avoid lock contention
	workerResponseTimes := make([][]float64, numWorkers)
	workerSamples := make([][]ResponseTimeSnapshot, numWorkers)
	for i := 0; i < numWorkers; i++ {
		workerResponseTimes[i] = make([]float64, 0, 10000/numWorkers)
		workerSamples[i] = make([]ResponseTimeSnapshot, 0, 1000/numWorkers)
	}

	return &Collector{
		config:              config,
		httpClient:          httpClient,
		workerResponseTimes: workerResponseTimes,
		workerSamples:       workerSamples,
		maxSamples:          1000,
	}
}

// workerStats holds statistics for a single worker
type workerStats struct {
	startTime time.Time
	endTime   time.Time
	requests  int64
}

// Run executes the request sending loop and returns collected data
func (c *Collector) Run(ctx context.Context) (*RequestData, error) {
	// Calculate QPS interval
	qps := c.config.QPS
	if qps <= 0 {
		qps = 1
	}

	targetURL := fmt.Sprintf("http://%s:%d/calculate", c.config.TargetIP, c.config.TargetPort)

	// Use WaitGroup to track worker goroutines
	var wg sync.WaitGroup

	// Create 16 parallel worker goroutines for better performance
	numWorkers := 16

	// Channel to collect worker statistics
	statsChan := make(chan workerStats, numWorkers)

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			var workerStart time.Time
			var workerEnd time.Time
			var requestCount int64

			// Calculate interval: multiply first to avoid integer division precision loss
			// interval = (1 second * numWorkers) / qps
			interval := (time.Second * time.Duration(numWorkers)) / time.Duration(qps)
			if interval <= 0 {
				interval = time.Microsecond
			}

			ticker := time.NewTicker(interval)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					// Send worker stats (using last request time, not context cancel time)
					if requestCount > 0 {
						statsChan <- workerStats{
							startTime: workerStart,
							endTime:   workerEnd,
							requests:  requestCount,
						}
					}
					return
				case <-ticker.C:
					// Record request time and launch request asynchronously
					requestTime := time.Now()
					if requestCount == 0 {
						workerStart = requestTime
					}
					workerEnd = requestTime
					requestCount++

					// Send request asynchronously to avoid blocking ticker
					go c.sendRequest(ctx, targetURL, workerID)
				}
			}
		}(i)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Wait for all workers to finish
	wg.Wait()
	close(statsChan)

	// Collect worker statistics and calculate average
	var totalDuration time.Duration
	var totalQPS float64
	workerCount := 0

	overallStart := time.Now()
	overallEnd := time.Time{}

	for stats := range statsChan {
		duration := stats.endTime.Sub(stats.startTime)
		totalDuration += duration

		if duration.Seconds() > 0 {
			workerQPS := float64(stats.requests) / duration.Seconds()
			totalQPS += workerQPS
		}

		// Track overall start and end
		if workerCount == 0 || stats.startTime.Before(overallStart) {
			overallStart = stats.startTime
		}
		if stats.endTime.After(overallEnd) {
			overallEnd = stats.endTime
		}

		workerCount++
	}

	// Use overall start and end time for result, but with accurate QPS from workers
	return c.buildResultData(overallStart, overallEnd, totalQPS), nil
}

// sendRequest sends a single HTTP request and records statistics
func (c *Collector) sendRequest(ctx context.Context, targetURL string, workerID int) {
	startTime := time.Now()

	// Create request with empty JSON body
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewBufferString("{}"))
	if err != nil {
		c.recordFailure(startTime, err, workerID)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	responseTime := time.Since(startTime)

	if err != nil {
		c.recordFailure(startTime, err, workerID)
		return
	}
	defer resp.Body.Close()

	// CRITICAL: Must read and discard response body to enable connection reuse
	// If body is not fully read, the connection will be closed instead of returned to the pool
	_, _ = io.Copy(io.Discard, resp.Body)

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.recordSuccess(startTime, responseTime, workerID)
	} else {
		c.recordFailure(startTime, fmt.Errorf("HTTP %d", resp.StatusCode), workerID)
	}
}

// recordSuccess records a successful request (lock-free per-worker collection)
func (c *Collector) recordSuccess(timestamp time.Time, responseTime time.Duration, workerID int) {
	c.totalRequests.Add(1)
	c.successful.Add(1)

	rtMs := float64(responseTime.Nanoseconds()) / 1e6

	// Store response time in worker-specific slice (no lock needed)
	c.workerResponseTimes[workerID] = append(c.workerResponseTimes[workerID], rtMs)

	// Store sample in worker-specific slice (limited, no lock needed)
	if len(c.workerSamples[workerID]) < c.maxSamples/16 {
		c.workerSamples[workerID] = append(c.workerSamples[workerID], ResponseTimeSnapshot{
			Timestamp:    timestamp,
			ResponseTime: rtMs,
			Success:      true,
		})
	}
}

// recordFailure records a failed request (lock-free per-worker collection)
func (c *Collector) recordFailure(timestamp time.Time, err error, workerID int) {
	c.totalRequests.Add(1)
	c.failed.Add(1)

	// Store sample in worker-specific slice (limited, no lock needed)
	if len(c.workerSamples[workerID]) < c.maxSamples/16 {
		c.workerSamples[workerID] = append(c.workerSamples[workerID], ResponseTimeSnapshot{
			Timestamp:    timestamp,
			ResponseTime: 0,
			Success:      false,
		})
	}
}

// buildResultData constructs the final RequestData from collected statistics
func (c *Collector) buildResultData(startTime, endTime time.Time, actualQPS float64) *RequestData {
	duration := endTime.Sub(startTime).Seconds()

	totalReqs := c.totalRequests.Load()
	successful := c.successful.Load()
	failed := c.failed.Load()

	// Calculate statistics (will merge worker response times internally)
	// Use actualQPS from per-worker timing instead of overall duration
	stats := c.calculateStats(duration, totalReqs, failed, actualQPS)

	// Merge all worker samples for response time snapshots
	var allSamples []ResponseTimeSnapshot
	for _, workerSamples := range c.workerSamples {
		allSamples = append(allSamples, workerSamples...)
	}

	return &RequestData{
		Config:        c.config,
		StartTime:     startTime,
		EndTime:       endTime,
		Duration:      duration,
		TotalRequests: totalReqs,
		Successful:    successful,
		Failed:        failed,
		Stats:         stats,
		ResponseTimes: allSamples,
	}
}

// calculateStats calculates statistical metrics from response times
func (c *Collector) calculateStats(duration float64, totalReqs, failed int64, actualQPS float64) RequestStats {
	stats := RequestStats{}

	// Merge all worker response times into a single slice
	var allResponseTimes []float64
	for _, workerTimes := range c.workerResponseTimes {
		allResponseTimes = append(allResponseTimes, workerTimes...)
	}

	if len(allResponseTimes) == 0 {
		stats.ErrorRate = 100.0
		// Use accurate QPS from per-worker timing
		stats.ActualQPS = actualQPS
		return stats
	}

	// Sort for percentile calculation
	sort.Float64s(allResponseTimes)

	// Calculate average
	var sum float64
	for _, rt := range allResponseTimes {
		sum += rt
	}
	stats.AvgResponseTime = sum / float64(len(allResponseTimes))

	// Min and Max
	stats.MinResponseTime = allResponseTimes[0]
	stats.MaxResponseTime = allResponseTimes[len(allResponseTimes)-1]

	// Percentiles
	stats.P50 = percentile(allResponseTimes, 0.5)
	stats.P95 = percentile(allResponseTimes, 0.95)
	stats.P99 = percentile(allResponseTimes, 0.99)

	// Error rate
	if totalReqs > 0 {
		stats.ErrorRate = float64(failed) / float64(totalReqs) * 100
	}

	// Use accurate QPS from per-worker timing (sum of all workers' QPS)
	// This avoids precision loss from overall start/end time differences
	stats.ActualQPS = actualQPS

	return stats
}

// percentile calculates the percentile value from a sorted slice
func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}

	index := float64(len(sorted)-1) * p
	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))

	if lower == upper {
		return sorted[lower]
	}

	// Linear interpolation
	weight := index - float64(lower)
	return sorted[lower]*(1-weight) + sorted[upper]*weight
}
