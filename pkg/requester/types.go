package requester

import (
	"encoding/json"
	"time"
)

// Config represents the configuration for a request experiment
type Config struct {
	TargetIP   string `json:"target_ip"`
	TargetPort int    `json:"target_port"`
	QPS        int    `json:"qps"`
	Timeout    int    `json:"timeout"` // in seconds
}

// RequestData represents the collected data from a request experiment
type RequestData struct {
	Config        Config                 `json:"config"`
	StartTime     time.Time              `json:"start_time"`
	EndTime       time.Time              `json:"end_time"`
	Duration      float64                `json:"duration"` // in seconds
	TotalRequests int64                  `json:"total_requests"`
	Successful    int64                  `json:"successful"`
	Failed        int64                  `json:"failed"`
	Stats         RequestStats           `json:"stats"`
	ResponseTimes []ResponseTimeSnapshot `json:"response_times,omitempty"` // Sample of response times
}

// RequestStats represents statistical data about requests
type RequestStats struct {
	AvgResponseTime float64 `json:"avg_response_time"` // in milliseconds
	MinResponseTime float64 `json:"min_response_time"` // in milliseconds
	MaxResponseTime float64 `json:"max_response_time"` // in milliseconds
	P50             float64 `json:"p50"`               // 50th percentile
	P95             float64 `json:"p95"`               // 95th percentile
	P99             float64 `json:"p99"`               // 99th percentile
	ErrorRate       float64 `json:"error_rate"`        // percentage
	ActualQPS       float64 `json:"actual_qps"`        // actual requests per second
}

// ResponseTimeSnapshot represents a sample of response time at a specific time
type ResponseTimeSnapshot struct {
	Timestamp    time.Time `json:"timestamp"`
	ResponseTime float64   `json:"response_time"` // in milliseconds
	Success      bool      `json:"success"`
}

// MarshalJSON implements json.Marshaler interface for RequestData
func (r RequestData) MarshalJSON() ([]byte, error) {
	type Alias RequestData
	return json.Marshal((Alias)(r))
}

// UnmarshalJSON implements json.Unmarshaler interface for RequestData
func (r *RequestData) UnmarshalJSON(data []byte) error {
	type Alias RequestData
	return json.Unmarshal(data, (*Alias)(r))
}
