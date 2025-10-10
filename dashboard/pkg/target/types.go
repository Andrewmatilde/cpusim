package target

import (
	"time"

	collectorAPI "cpusim/collector/api/generated"
)

// Target represents a target host that runs cpusim-server and collector-server
type Target struct {
	Name       string
	ExternalIP string
	InternalIP string
}

// GetCPUServiceURL returns the CPU service URL for this target
func (t *Target) GetCPUServiceURL() string {
	return "http://" + t.ExternalIP + ":80"
}

// GetCollectorServiceURL returns the collector service URL for this target
func (t *Target) GetCollectorServiceURL() string {
	return "http://" + t.ExternalIP + ":8080"
}

// TargetHealth represents the health status of a target host
type TargetHealth struct {
	Name                    string
	ExternalIP              string
	InternalIP              string
	CPUServiceHealthy       bool
	CollectorServiceHealthy bool
	LastChecked             time.Time
}

// CollectorData represents data collected from a target's collector service
type CollectorData struct {
	Duration int
	Metrics  []collectorAPI.MetricDataPoint
}