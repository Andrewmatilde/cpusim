package collector

import (
	"encoding/json"
	"time"
)

// Config defines the collector configuration
type Config struct {
	CollectionInterval int    `json:"collection_interval"` // in seconds
	CalculatorProcess  string `json:"calculator_process"`  // process name to monitor
}

// MetricsData contains all collected metrics for an experiment
type MetricsData struct {
	Config             Config            `json:"config"`
	StartTime          time.Time         `json:"start_time"`
	EndTime            time.Time         `json:"end_time"`
	Duration           float64           `json:"duration"` // in seconds
	DataPointsCollected int              `json:"data_points_collected"`
	Metrics            []MetricDataPoint `json:"metrics"`
}

// MetricDataPoint represents a single measurement point
type MetricDataPoint struct {
	Timestamp                time.Time `json:"timestamp"`
	CPUUsagePercent          float64   `json:"cpu_usage_percent"`
	MemoryUsageBytes         int64     `json:"memory_usage_bytes"`
	MemoryUsagePercent       float64   `json:"memory_usage_percent"`
	NetworkIOBytes           NetworkIO `json:"network_io_bytes"`
	CalculatorServiceHealthy bool      `json:"calculator_service_healthy"`
}

// NetworkIO represents network I/O statistics
type NetworkIO struct {
	BytesReceived   int64 `json:"bytes_received"`
	BytesSent       int64 `json:"bytes_sent"`
	PacketsReceived int64 `json:"packets_received"`
	PacketsSent     int64 `json:"packets_sent"`
}

// Implement json.Marshaler and json.Unmarshaler for MetricsData
func (m MetricsData) MarshalJSON() ([]byte, error) {
	type Alias MetricsData
	return json.Marshal((Alias)(m))
}

func (m *MetricsData) UnmarshalJSON(data []byte) error {
	type Alias MetricsData
	return json.Unmarshal(data, (*Alias)(m))
}
