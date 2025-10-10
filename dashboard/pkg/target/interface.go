package target

import (
	"context"

	collectorAPI "cpusim/collector/api/generated"
	dashboardAPI "cpusim/dashboard/api/generated"
)

// Manager defines the interface for managing target hosts
type Manager interface {
	// ListTargets returns all configured target hosts
	ListTargets() []*Target

	// GetTarget returns a target by name
	GetTarget(name string) (*Target, error)

	// CheckHealth checks the health of a specific target
	CheckHealth(ctx context.Context, targetName string) (*TargetHealth, error)

	// CheckAllHealth checks the health of all targets
	CheckAllHealth(ctx context.Context) (map[string]*TargetHealth, error)

	// StartCollector starts the collector service on a target
	StartCollector(ctx context.Context, targetName, experimentID, description string) error

	// StopCollector stops the collector service on a target and retrieves data
	StopCollector(ctx context.Context, targetName, experimentID string) (*CollectorData, error)

	// GetCollectorData retrieves collected data from a target
	GetCollectorData(ctx context.Context, targetName, experimentID string) (*CollectorData, error)

	// GetCollectorExperimentData retrieves experiment data in API format
	GetCollectorExperimentData(ctx context.Context, targetName, experimentID string) (*collectorAPI.ExperimentData, error)

	// TestCalculation tests the CPU calculation service on a target
	TestCalculation(ctx context.Context, targetName string, req dashboardAPI.CalculationRequest) (*dashboardAPI.CalculationResponse, error)
}