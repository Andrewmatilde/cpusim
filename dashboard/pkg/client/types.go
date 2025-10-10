package client

import (
	"time"
)

// Client represents a client host that runs requester-server
type Client struct {
	Name       string
	ExternalIP string
	InternalIP string
}

// GetRequesterServiceURL returns the requester service URL for this client
func (c *Client) GetRequesterServiceURL() string {
	return "http://" + c.ExternalIP + ":80"
}

// ClientHealth represents the health status of a client host
type ClientHealth struct {
	Name                    string
	ExternalIP              string
	InternalIP              string
	RequesterServiceHealthy bool
	LastChecked             time.Time
}

// RequesterConfig contains configuration for starting a requester
type RequesterConfig struct {
	ExperimentID   string
	TargetURL      string
	QPS            int
	Duration       int
	RequestTimeout int
}

// RequesterData represents data from a requester experiment
type RequesterData struct {
	TotalRequests       int
	SuccessfulRequests  int
	FailedRequests      int
	AverageResponseTime float32
	StartTime           time.Time
	EndTime             *time.Time
	Percentiles         *struct {
		P50 *float32
		P95 *float32
		P99 *float32
	}
}