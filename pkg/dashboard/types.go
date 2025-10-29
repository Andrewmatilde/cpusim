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

	// Load balancer configuration (optional)
	LoadBalancer *LoadBalancer `json:"load_balancer,omitempty"`
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

// LoadBalancer represents the load balancer between client and target hosts
type LoadBalancer struct {
	Name       string `json:"name"`
	ExternalIP string `json:"external_ip"`
	InternalIP string `json:"internal_ip"`

	// Service URL (if LB provides metrics/status endpoint)
	ServiceURL string `json:"service_url,omitempty"`
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

// ExperimentGroup represents a group of experiments across QPS range
type ExperimentGroup struct {
	GroupID           string                `json:"group_id"`
	Description       string                `json:"description,omitempty"`
	Config            ExperimentGroupConfig `json:"config"`
	EnvironmentConfig Config                `json:"environment_config"` // Client and Target host information
	QPSPoints         []QPSPoint            `json:"qps_points"`         // Organized by QPS value
	StartTime         time.Time             `json:"start_time"`
	EndTime           time.Time             `json:"end_time,omitempty"`
	Status            string                `json:"status"`      // "running", "completed", "failed"
	CurrentQPS        int                   `json:"current_qps"` // Current QPS being tested
	CurrentRun        int                   `json:"current_run"` // Current run for current QPS (1-based)
}

// QPSPoint represents results for a specific QPS value
type QPSPoint struct {
	QPS              int                       `json:"qps"`               // QPS value for this point
	Experiments      []string                  `json:"experiments"`       // List of experiment IDs for this QPS
	Statistics       map[string]*CPUStats      `json:"statistics"`        // CPU stats per host (key: host name)
	LatencyStats     *LatencyStats             `json:"latency_stats"`     // Global latency stats from requester
	Status           string                    `json:"status"`            // "running", "completed", "failed"
}

// CPUStats contains CPU performance statistics with confidence intervals for a specific host
type CPUStats struct {
	CPUMean         float64 `json:"cpu_mean"`         // Mean CPU usage across all experiments
	CPUStdDev       float64 `json:"cpu_std_dev"`      // Standard deviation
	CPUConfLower    float64 `json:"cpu_conf_lower"`   // 95% CI lower bound
	CPUConfUpper    float64 `json:"cpu_conf_upper"`   // 95% CI upper bound
	CPUMin          float64 `json:"cpu_min"`          // Minimum value
	CPUMax          float64 `json:"cpu_max"`          // Maximum value
	SampleSize      int     `json:"sample_size"`      // Number of experiments used
	ConfidenceLevel float64 `json:"confidence_level"` // Confidence level (e.g., 0.95)
}

// LatencyStats contains latency performance statistics from requester perspective
type LatencyStats struct {
	LatencyP50  float64 `json:"latency_p50"`  // Median latency in milliseconds
	LatencyP90  float64 `json:"latency_p90"`  // 90th percentile latency
	LatencyP95  float64 `json:"latency_p95"`  // 95th percentile latency
	LatencyP99  float64 `json:"latency_p99"`  // 99th percentile latency
	LatencyMean float64 `json:"latency_mean"` // Mean latency
	LatencyMin  float64 `json:"latency_min"`  // Min latency
	LatencyMax  float64 `json:"latency_max"`  // Max latency
	Throughput  float64 `json:"throughput"`   // Successful requests per second
	ErrorRate   float64 `json:"error_rate"`   // Error rate percentage
	Utilization float64 `json:"utilization"`  // Server utilization (λ/μ)
	SampleSize  int     `json:"sample_size"`  // Number of experiments used
}

// SteadyStateStats contains steady-state performance statistics with confidence intervals
// Deprecated: Use CPUStats and LatencyStats separately instead
type SteadyStateStats struct {
	// CPU statistics
	CPUMean      float64 `json:"cpu_mean"`       // Mean CPU usage across all experiments
	CPUStdDev    float64 `json:"cpu_std_dev"`    // Standard deviation
	CPUConfLower float64 `json:"cpu_conf_lower"` // 95% CI lower bound
	CPUConfUpper float64 `json:"cpu_conf_upper"` // 95% CI upper bound
	CPUMin       float64 `json:"cpu_min"`        // Minimum value
	CPUMax       float64 `json:"cpu_max"`        // Maximum value

	// Latency statistics (from requester)
	LatencyP50   float64 `json:"latency_p50"`    // Median latency in milliseconds
	LatencyP90   float64 `json:"latency_p90"`    // 90th percentile latency
	LatencyP95   float64 `json:"latency_p95"`    // 95th percentile latency
	LatencyP99   float64 `json:"latency_p99"`    // 99th percentile latency
	LatencyMean  float64 `json:"latency_mean"`   // Mean latency
	LatencyMin   float64 `json:"latency_min"`    // Min latency
	LatencyMax   float64 `json:"latency_max"`    // Max latency
	Throughput   float64 `json:"throughput"`     // Successful requests per second
	ErrorRate    float64 `json:"error_rate"`     // Error rate percentage
	Utilization  float64 `json:"utilization"`    // Server utilization (λ/μ)

	SampleSize      int     `json:"sample_size"`      // Number of experiments used
	ConfidenceLevel float64 `json:"confidence_level"` // Confidence level (e.g., 0.95)
}

// ExperimentGroupConfig defines the configuration for an experiment group
type ExperimentGroupConfig struct {
	QPSMin       int `json:"qps_min"`       // Minimum QPS value (e.g., 100)
	QPSMax       int `json:"qps_max"`       // Maximum QPS value (e.g., 500)
	QPSStep      int `json:"qps_step"`      // Step size for QPS values (e.g., 100)
	RepeatCount  int `json:"repeat_count"`  // Number of times to repeat each QPS
	Timeout      int `json:"timeout"`       // Timeout for each experiment in seconds
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
