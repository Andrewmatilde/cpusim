package requester

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
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

	// Configure HTTP transport for connection pooling with keep-alive
	// Uses persistent connections to reduce connection overhead
	transport := &http.Transport{
		MaxIdleConns:        200,  // Maximum idle connections across all hosts
		MaxIdleConnsPerHost: 100,  // Maximum idle connections per host
		MaxConnsPerHost:     200,  // Maximum connections per host (including active)
		IdleConnTimeout:     90 * time.Second, // Keep idle connections alive
		DisableKeepAlives:   false, // Enable HTTP keep-alive for connection reuse
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

	// OPTIMIZATION: Use buffered channel as request queue to avoid creating goroutines on every tick
	// Each worker will have its own queue to maintain rate limiting per worker
	// Buffer must be large enough to hold all requests for the duration of the experiment
	// At 1400 QPS for 60s = 84000 total / 16 workers = 5250 per worker
	// Use 10000 to be safe and handle bursts
	requestQueues := make([]chan struct{}, numWorkers)
	for i := 0; i < numWorkers; i++ {
		requestQueues[i] = make(chan struct{}, 10000)
	}

	// Start request sender goroutines (one per worker, reused for all requests)
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			queue := requestQueues[workerID]

			for {
				select {
				case <-ctx.Done():
					return
				case <-queue:
					// Send request synchronously in this dedicated goroutine
					c.sendRequest(ctx, targetURL, workerID)
				}
			}
		}(i)
	}

	// Start ticker goroutines (one per worker, controls rate)
	// Support both uniform and Poisson arrival patterns
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			var workerStart time.Time
			var workerEnd time.Time
			var requestCount int64

			// Calculate base interval for this worker
			// interval = (1 second * numWorkers) / qps
			baseInterval := (time.Second * time.Duration(numWorkers)) / time.Duration(qps)
			if baseInterval <= 0 {
				baseInterval = time.Microsecond
			}

			queue := requestQueues[workerID]

			// Use Poisson arrival if configured, otherwise uniform
			usePoisson := c.config.ArrivalPattern == ArrivalPatternPoisson

			// For Poisson: lambda (rate parameter) per worker
			lambda := float64(qps) / float64(numWorkers)

			// Initialize random source for Poisson process (per-worker to avoid lock contention)
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerID)))

			var ticker *time.Ticker
			var timer *time.Timer
			var nextEventTime time.Time

			if !usePoisson {
				// Uniform: use ticker
				ticker = time.NewTicker(baseInterval)
				defer ticker.Stop()
			} else {
				// Poisson: start from current time
				nextEventTime = time.Now()
			}

			for {
				if usePoisson {
					// Poisson arrival: base next event on planned time, not actual time
					// This prevents "catch-up" bursts after blocking/GC pauses
					nextEventTime = nextEventTime.Add(c.exponentialDelay(lambda, rng))
					waitDuration := time.Until(nextEventTime)
					if waitDuration < 0 {
						waitDuration = 0 // Allow slight jitter but don't accumulate multiple events
					}

					// Reuse timer to reduce allocations
					if timer == nil {
						timer = time.NewTimer(waitDuration)
					} else {
						if !timer.Stop() {
							select {
							case <-timer.C:
							default:
							}
						}
						timer.Reset(waitDuration)
					}

					select {
					case <-ctx.Done():
						if timer != nil && !timer.Stop() {
							select {
							case <-timer.C:
							default:
							}
						}
						if requestCount > 0 {
							statsChan <- workerStats{
								startTime: workerStart,
								endTime:   workerEnd,
								requests:  requestCount,
							}
						}
						return

					case <-timer.C:
						// Record arrival time (for arrival process statistics)
						now := time.Now()
						if requestCount == 0 {
							workerStart = now
						}
						workerEnd = now
						requestCount++

						// Send to queue
						select {
						case queue <- struct{}{}:
							// Queued successfully
						case <-ctx.Done():
							return
						}
					}
				} else {
					// Uniform arrival: use ticker
					select {
					case <-ctx.Done():
						if requestCount > 0 {
							statsChan <- workerStats{
								startTime: workerStart,
								endTime:   workerEnd,
								requests:  requestCount,
							}
						}
						return
					case <-ticker.C:
						// Record request time and queue request
						requestTime := time.Now()
						if requestCount == 0 {
							workerStart = requestTime
						}
						workerEnd = requestTime
						requestCount++

						// Send to queue with context check to prevent blocking forever
						select {
						case queue <- struct{}{}:
							// Queued successfully
						case <-ctx.Done():
							// Context cancelled while trying to queue
							return
						}
					}
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
	stats.P90 = percentile(allResponseTimes, 0.90)
	stats.P95 = percentile(allResponseTimes, 0.95)
	stats.P99 = percentile(allResponseTimes, 0.99)

	// Error rate
	if totalReqs > 0 {
		stats.ErrorRate = float64(failed) / float64(totalReqs) * 100
	}

	// Use accurate QPS from per-worker timing (sum of all workers' QPS)
	// This avoids precision loss from overall start/end time differences
	stats.ActualQPS = actualQPS

	// Calculate latency buckets (histogram)
	stats.LatencyBuckets = c.calculateLatencyBuckets(allResponseTimes)

	// Calculate queueing theory metrics
	successfulReqs := totalReqs - failed
	if successfulReqs > 0 && duration > 0 {
		// Throughput: successful requests per second
		stats.Throughput = float64(successfulReqs) / duration

		// Utilization: λ/μ where λ is arrival rate and μ is service rate
		// λ (lambda) = actual QPS (arrival rate)
		// μ (mu) = 1 / average response time (service rate)
		// Average response time in seconds
		avgResponseTimeSec := stats.AvgResponseTime / 1000.0
		if avgResponseTimeSec > 0 {
			serviceRate := 1.0 / avgResponseTimeSec // μ
			stats.Utilization = actualQPS / serviceRate
		}
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

// exponentialDelay generates a random delay following exponential distribution
// for Poisson arrival process. Lambda is the arrival rate (events per second).
// Returns inter-arrival time as duration.
func (c *Collector) exponentialDelay(lambda float64, rng *rand.Rand) time.Duration {
	// For Poisson process, inter-arrival times follow exponential distribution
	// E[X] = 1/lambda (mean inter-arrival time)
	// Using inverse transform: X = -ln(U)/lambda where U ~ Uniform(0,1)
	u := rng.Float64()
	if u == 0 {
		u = 1e-10 // Avoid log(0)
	}
	delaySeconds := -math.Log(u) / lambda
	return time.Duration(delaySeconds * float64(time.Second))
}

// calculateLatencyBuckets creates a histogram of latency distribution
// Buckets: <10ms, 10-50ms, 50-100ms, 100-200ms, 200-500ms, 500ms-1s, 1s-2s, >2s
func (c *Collector) calculateLatencyBuckets(responseTimes []float64) map[string]int64 {
	buckets := map[string]int64{
		"<10ms":      0,
		"10-50ms":    0,
		"50-100ms":   0,
		"100-200ms":  0,
		"200-500ms":  0,
		"500ms-1s":   0,
		"1s-2s":      0,
		">2s":        0,
	}

	for _, rt := range responseTimes {
		switch {
		case rt < 10:
			buckets["<10ms"]++
		case rt < 50:
			buckets["10-50ms"]++
		case rt < 100:
			buckets["50-100ms"]++
		case rt < 200:
			buckets["100-200ms"]++
		case rt < 500:
			buckets["200-500ms"]++
		case rt < 1000:
			buckets["500ms-1s"]++
		case rt < 2000:
			buckets["1s-2s"]++
		default:
			buckets[">2s"]++
		}
	}

	return buckets
}
