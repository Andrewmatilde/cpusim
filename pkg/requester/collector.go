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

// Run executes the request sending loop and returns collected data
func (c *Collector) Run(ctx context.Context) (*RequestData, error) {
	startTime := time.Now()

	// Calculate QPS interval
	qps := c.config.QPS
	if qps <= 0 {
		qps = 1
	}
	interval := time.Second / time.Duration(qps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	targetURL := fmt.Sprintf("http://%s:%d/calculate", c.config.TargetIP, c.config.TargetPort)

	// Use WaitGroup to track in-flight requests
	var wg sync.WaitGroup

	// Request sending loop
	for {
		select {
		case <-ctx.Done():
			// Wait for all in-flight requests to complete
			wg.Wait()

			// Calculate final statistics
			endTime := time.Now()
			return c.buildResultData(startTime, endTime), nil

		case <-ticker.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				c.sendRequest(ctx, targetURL)
			}()
		}
	}
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
