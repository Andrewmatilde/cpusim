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

	// Response times for statistics calculation
	responseTimes []float64
	rtMu          sync.Mutex

	// Detailed samples (limited to avoid memory issues)
	samples    []ResponseTimeSnapshot
	samplesMu  sync.Mutex
	maxSamples int
}

// NewCollector creates a new request collector
func NewCollector(config Config) *Collector {
	// Configure HTTP transport for high concurrency
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	return &Collector{
		config:        config,
		httpClient:    httpClient,
		responseTimes: make([]float64, 0, 10000),
		samples:       make([]ResponseTimeSnapshot, 0, 1000),
		maxSamples:    1000,
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
		go func() {
			defer wg.Done()

			var workerStart time.Time
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
					// Record worker end time
					workerEnd := time.Now()
					if requestCount > 0 {
						statsChan <- workerStats{
							startTime: workerStart,
							endTime:   workerEnd,
							requests:  requestCount,
						}
					}
					return
				case <-ticker.C:
					// Record first request time
					if requestCount == 0 {
						workerStart = time.Now()
					}
					c.sendRequest(ctx, targetURL)
					requestCount++
				}
			}
		}()
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

	// Use overall start and end time for result
	return c.buildResultData(overallStart, overallEnd), nil
}

// sendRequest sends a single HTTP request and records statistics
func (c *Collector) sendRequest(ctx context.Context, targetURL string) {
	startTime := time.Now()

	// Create request with empty JSON body
	req, err := http.NewRequestWithContext(ctx, "POST", targetURL, bytes.NewBufferString("{}"))
	if err != nil {
		c.recordFailure(startTime, err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	responseTime := time.Since(startTime)

	if err != nil {
		c.recordFailure(startTime, err)
		return
	}
	defer resp.Body.Close()

	// CRITICAL: Must read and discard response body to enable connection reuse
	// If body is not fully read, the connection will be closed instead of returned to the pool
	_, _ = io.Copy(io.Discard, resp.Body)

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.recordSuccess(startTime, responseTime)
	} else {
		c.recordFailure(startTime, fmt.Errorf("HTTP %d", resp.StatusCode))
	}
}

// recordSuccess records a successful request
func (c *Collector) recordSuccess(timestamp time.Time, responseTime time.Duration) {
	c.totalRequests.Add(1)
	c.successful.Add(1)

	rtMs := float64(responseTime.Nanoseconds()) / 1e6

	// Store response time for statistics
	c.rtMu.Lock()
	c.responseTimes = append(c.responseTimes, rtMs)
	c.rtMu.Unlock()

	// Store sample (limited)
	c.samplesMu.Lock()
	if len(c.samples) < c.maxSamples {
		c.samples = append(c.samples, ResponseTimeSnapshot{
			Timestamp:    timestamp,
			ResponseTime: rtMs,
			Success:      true,
		})
	}
	c.samplesMu.Unlock()
}

// recordFailure records a failed request
func (c *Collector) recordFailure(timestamp time.Time, err error) {
	c.totalRequests.Add(1)
	c.failed.Add(1)

	// Store sample (limited)
	c.samplesMu.Lock()
	if len(c.samples) < c.maxSamples {
		c.samples = append(c.samples, ResponseTimeSnapshot{
			Timestamp:    timestamp,
			ResponseTime: 0,
			Success:      false,
		})
	}
	c.samplesMu.Unlock()
}

// buildResultData constructs the final RequestData from collected statistics
func (c *Collector) buildResultData(startTime, endTime time.Time) *RequestData {
	duration := endTime.Sub(startTime).Seconds()

	totalReqs := c.totalRequests.Load()
	successful := c.successful.Load()
	failed := c.failed.Load()

	// Calculate statistics
	stats := c.calculateStats(duration, totalReqs, failed)

	return &RequestData{
		Config:        c.config,
		StartTime:     startTime,
		EndTime:       endTime,
		Duration:      duration,
		TotalRequests: totalReqs,
		Successful:    successful,
		Failed:        failed,
		Stats:         stats,
		ResponseTimes: c.samples,
	}
}

// calculateStats calculates statistical metrics from response times
func (c *Collector) calculateStats(duration float64, totalReqs, failed int64) RequestStats {
	c.rtMu.Lock()
	defer c.rtMu.Unlock()

	stats := RequestStats{}

	if len(c.responseTimes) == 0 {
		stats.ErrorRate = 100.0
		if duration > 0 {
			stats.ActualQPS = float64(totalReqs) / duration
		}
		return stats
	}

	// Sort for percentile calculation
	sorted := make([]float64, len(c.responseTimes))
	copy(sorted, c.responseTimes)
	sort.Float64s(sorted)

	// Calculate average
	var sum float64
	for _, rt := range sorted {
		sum += rt
	}
	stats.AvgResponseTime = sum / float64(len(sorted))

	// Min and Max
	stats.MinResponseTime = sorted[0]
	stats.MaxResponseTime = sorted[len(sorted)-1]

	// Percentiles
	stats.P50 = percentile(sorted, 0.5)
	stats.P95 = percentile(sorted, 0.95)
	stats.P99 = percentile(sorted, 0.99)

	// Error rate
	if totalReqs > 0 {
		stats.ErrorRate = float64(failed) / float64(totalReqs) * 100
	}

	// Actual QPS
	if duration > 0 {
		stats.ActualQPS = float64(totalReqs) / duration
	}

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
