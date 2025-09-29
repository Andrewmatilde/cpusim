package experiment

import (
	"math"
	"sort"
	"sync"
	"time"

	"cpusim/requester/api/generated"
)

// RequestStats manages statistics for an experiment
type RequestStats struct {
	mu sync.RWMutex

	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	ResponseTimes      []float64
	LastUpdated        time.Time
}

// NewRequestStats creates a new request statistics tracker
func NewRequestStats() *RequestStats {
	return &RequestStats{
		ResponseTimes: make([]float64, 0),
		LastUpdated:   time.Now(),
	}
}

// RecordRequest records the result of a single request
func (s *RequestStats) RecordRequest(duration time.Duration, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.TotalRequests++
	s.LastUpdated = time.Now()

	if err != nil {
		s.FailedRequests++
	} else {
		s.SuccessfulRequests++
		// Convert duration to milliseconds
		s.ResponseTimes = append(s.ResponseTimes, float64(duration.Nanoseconds())/1e6)
	}
}

// CalculatePercentiles calculates response time percentiles
func (s *RequestStats) CalculatePercentiles() (p50, p95, p99 float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.ResponseTimes) == 0 {
		return 0, 0, 0
	}

	// Create a sorted copy of response times
	sorted := make([]float64, len(s.ResponseTimes))
	copy(sorted, s.ResponseTimes)
	sort.Float64s(sorted)

	p50 = percentile(sorted, 0.5)
	p95 = percentile(sorted, 0.95)
	p99 = percentile(sorted, 0.99)

	return p50, p95, p99
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

// GetAverageResponseTime calculates the average response time
func (s *RequestStats) GetAverageResponseTime() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.ResponseTimes) == 0 {
		return 0
	}

	var total float64
	for _, t := range s.ResponseTimes {
		total += t
	}

	return total / float64(len(s.ResponseTimes))
}

// GetMinMaxResponseTime gets the minimum and maximum response times
func (s *RequestStats) GetMinMaxResponseTime() (min, max float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.ResponseTimes) == 0 {
		return 0, 0
	}

	min = s.ResponseTimes[0]
	max = s.ResponseTimes[0]

	for _, t := range s.ResponseTimes {
		if t < min {
			min = t
		}
		if t > max {
			max = t
		}
	}

	return min, max
}

// GetErrorRate calculates the error rate as a percentage
func (s *RequestStats) GetErrorRate() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.TotalRequests == 0 {
		return 0
	}

	return float64(s.FailedRequests) / float64(s.TotalRequests) * 100
}

// GetRequestsPerSecond calculates the actual requests per second
func (s *RequestStats) GetRequestsPerSecond(duration time.Duration) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if duration.Seconds() <= 0 {
		return 0
	}

	return float64(s.TotalRequests) / duration.Seconds()
}

// ToRequestExperimentStats converts to API response format
func (s *RequestStats) ToRequestExperimentStats(experimentId, status string, startTime time.Time, endTime *time.Time) *generated.RequestExperimentStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Calculate percentiles
	p50, p95, p99 := s.CalculatePercentiles()

	// Calculate average, min, max
	avgResponseTime := s.GetAverageResponseTime()
	minResponseTime, maxResponseTime := s.GetMinMaxResponseTime()

	// Calculate duration and QPS
	var duration int
	var qps float64
	if endTime != nil {
		duration = int(endTime.Sub(startTime).Seconds())
		qps = s.GetRequestsPerSecond(endTime.Sub(startTime))
	} else {
		duration = int(time.Since(startTime).Seconds())
		qps = s.GetRequestsPerSecond(time.Since(startTime))
	}

	// Calculate error rate
	errorRate := s.GetErrorRate()

	// Convert to the correct types
	var statsStatus generated.RequestExperimentStatsStatus
	switch status {
	case "running":
		statsStatus = generated.RequestExperimentStatsStatusRunning
	case "stopped":
		statsStatus = generated.RequestExperimentStatsStatusStopped
	case "completed":
		statsStatus = generated.RequestExperimentStatsStatusCompleted
	case "error":
		statsStatus = generated.RequestExperimentStatsStatusError
	}

	// Convert int64 to int
	totalRequests := int(s.TotalRequests)
	successfulRequests := int(s.SuccessfulRequests)
	failedRequests := int(s.FailedRequests)

	// Convert float64 to float32
	avgResponseTimeF32 := float32(avgResponseTime)
	minResponseTimeF32 := float32(minResponseTime)
	maxResponseTimeF32 := float32(maxResponseTime)
	qpsF32 := float32(qps)
	errorRateF32 := float32(errorRate)
	p50F32 := float32(p50)
	p95F32 := float32(p95)
	p99F32 := float32(p99)

	return &generated.RequestExperimentStats{
		ExperimentId:        &experimentId,
		Status:              &statsStatus,
		TotalRequests:       &totalRequests,
		SuccessfulRequests:  &successfulRequests,
		FailedRequests:      &failedRequests,
		AverageResponseTime: &avgResponseTimeF32,
		MinResponseTime:     &minResponseTimeF32,
		MaxResponseTime:     &maxResponseTimeF32,
		RequestsPerSecond:   &qpsF32,
		ErrorRate:           &errorRateF32,
		ResponseTimeP50:     &p50F32,
		ResponseTimeP95:     &p95F32,
		ResponseTimeP99:     &p99F32,
		StartTime:           &startTime,
		EndTime:             endTime,
		Duration:            &duration,
		LastUpdated:         &s.LastUpdated,
	}
}

// GetSnapshot returns a snapshot of current statistics
func (s *RequestStats) GetSnapshot() RequestStatsSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return RequestStatsSnapshot{
		TotalRequests:      s.TotalRequests,
		SuccessfulRequests: s.SuccessfulRequests,
		FailedRequests:     s.FailedRequests,
		ResponseTimeCount:  len(s.ResponseTimes),
		LastUpdated:        s.LastUpdated,
	}
}

// RequestStatsSnapshot represents a snapshot of statistics without holding the full response time data
type RequestStatsSnapshot struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	ResponseTimeCount  int
	LastUpdated        time.Time
}