package dashboard

import (
	"encoding/json"
	"time"

	collectorAPI "cpusim/collector/api/generated"
	requesterAPI "cpusim/requester/api/generated"
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
	Config    Config    `json:"config"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Duration  float64   `json:"duration"` // seconds
	Status    string    `json:"status"`   // "completed", "failed", "partial"

	// Sub-experiment results
	CollectorResults map[string]CollectorResult `json:"collector_results"` // key: target host name
	RequesterResult  *RequesterResult           `json:"requester_result"`

	// Error tracking
	Errors []ExperimentError `json:"errors,omitempty"`
}

// CollectorResult stores the result from a collector experiment
type CollectorResult struct {
	HostName string `json:"host_name"`
	Status   string `json:"status"` // "completed", "failed", "not_started"
	Error    string `json:"error,omitempty"`

	// Complete experiment data from collector
	Data *collectorAPI.ExperimentData `json:"data,omitempty"`
}

// RequesterResult stores the result from the requester experiment
type RequesterResult struct {
	Status string `json:"status"` // "completed", "failed", "not_started"
	Error  string `json:"error,omitempty"`

	// Complete experiment stats from requester
	Stats *requesterAPI.RequestExperimentStats `json:"stats,omitempty"`
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

// ExperimentGroup represents a group of repeated experiments
type ExperimentGroup struct {
	GroupID     string                `json:"group_id"`
	Description string                `json:"description,omitempty"`
	Config      ExperimentGroupConfig `json:"config"`
	Experiments []string              `json:"experiments"` // List of experiment IDs
	StartTime   time.Time             `json:"start_time"`
	EndTime     time.Time             `json:"end_time,omitempty"`
	Status      string                `json:"status"`      // "running", "completed", "failed"
	CurrentRun  int                   `json:"current_run"` // 1-based, current execution number

	// Steady-state statistics (calculated per host)
	Statistics map[string]*SteadyStateStats `json:"statistics,omitempty"` // key: host name
}

// SteadyStateStats contains steady-state performance statistics with confidence intervals
type SteadyStateStats struct {
	// CPU statistics
	CPUMean      float64 `json:"cpu_mean"`       // Mean CPU usage across all experiments
	CPUStdDev    float64 `json:"cpu_std_dev"`    // Standard deviation
	CPUConfLower float64 `json:"cpu_conf_lower"` // 95% CI lower bound
	CPUConfUpper float64 `json:"cpu_conf_upper"` // 95% CI upper bound
	CPUMin       float64 `json:"cpu_min"`        // Minimum value
	CPUMax       float64 `json:"cpu_max"`        // Maximum value

	SampleSize      int     `json:"sample_size"`      // Number of experiments used
	ConfidenceLevel float64 `json:"confidence_level"` // Confidence level (e.g., 0.95)
}

// ExperimentGroupConfig defines the configuration for an experiment group
type ExperimentGroupConfig struct {
	RepeatCount  int `json:"repeat_count"`  // Number of times to repeat
	Timeout      int `json:"timeout"`       // Timeout for each experiment in seconds
	QPS          int `json:"qps"`           // QPS for each experiment
	DelayBetween int `json:"delay_between"` // Delay between experiments in seconds
}

// Implement json.Marshaler and json.Unmarshaler for ExperimentGroup
func (g ExperimentGroup) MarshalJSON() ([]byte, error) {
	type Alias ExperimentGroup
	return json.Marshal((Alias)(g))
}

func (g *ExperimentGroup) UnmarshalJSON(data []byte) error {
	type Alias ExperimentGroup
	return json.Unmarshal(data, (*Alias)(g))
}
