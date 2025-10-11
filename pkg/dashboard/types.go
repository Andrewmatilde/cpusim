package dashboard

import (
	"encoding/json"
	"time"
)

// Config defines the dashboard service configuration
// Dashboard only needs to know service URLs, not their runtime configs
type Config struct {
	// Target hosts configuration
	TargetHosts []TargetHost `json:"target_hosts"`

	// Client host configuration
	ClientHost ClientHost `json:"client_host"`
}

// TargetHost represents a target server to collect metrics from
type TargetHost struct {
	Name       string `json:"name"`
	ExternalIP string `json:"external_ip"`
	InternalIP string `json:"internal_ip"`

	// Service URLs
	CPUServiceURL       string `json:"cpu_service_url"`
	CollectorServiceURL string `json:"collector_service_url"`
}

// ClientHost represents the client that sends requests
type ClientHost struct {
	Name       string `json:"name"`
	ExternalIP string `json:"external_ip"`
	InternalIP string `json:"internal_ip"`

	// Service URL
	RequesterServiceURL string `json:"requester_service_url"`
}

// ExperimentData contains the complete dashboard experiment result
type ExperimentData struct {
	Config      Config                   `json:"config"`
	StartTime   time.Time                `json:"start_time"`
	EndTime     time.Time                `json:"end_time"`
	Duration    float64                  `json:"duration"` // seconds
	Status      string                   `json:"status"`   // "completed", "failed", "partial"

	// Sub-experiment results
	CollectorResults map[string]CollectorResult `json:"collector_results"` // key: target host name
	RequesterResult  *RequesterResult           `json:"requester_result"`

	// Error tracking
	Errors []ExperimentError `json:"errors,omitempty"`
}

// CollectorResult stores the result from a collector experiment
type CollectorResult struct {
	HostName          string    `json:"host_name"`
	ExperimentID      string    `json:"experiment_id"`
	Status            string    `json:"status"` // "completed", "failed", "not_started"
	DataPointsCollected int     `json:"data_points_collected"`
	Error             string    `json:"error,omitempty"`
}

// RequesterResult stores the result from the requester experiment
type RequesterResult struct {
	ExperimentID    string  `json:"experiment_id"`
	Status          string  `json:"status"` // "completed", "failed", "not_started"
	TotalRequests   int64   `json:"total_requests"`
	Successful      int64   `json:"successful"`
	Failed          int64   `json:"failed"`
	AvgResponseTime float64 `json:"avg_response_time"`
	Error           string  `json:"error,omitempty"`
}

// ExperimentError records errors that occurred during the experiment
type ExperimentError struct {
	Timestamp time.Time `json:"timestamp"`
	Phase     string    `json:"phase"` // "collector_start", "requester_start", "stop", etc.
	HostName  string    `json:"host_name,omitempty"`
	Message   string    `json:"message"`
}

// Implement json.Marshaler and json.Unmarshaler for ExperimentData
func (e ExperimentData) MarshalJSON() ([]byte, error) {
	type Alias ExperimentData
	return json.Marshal((Alias)(e))
}

func (e *ExperimentData) UnmarshalJSON(data []byte) error {
	type Alias ExperimentData
	return json.Unmarshal(data, (*Alias)(e))
}
