package experiment

import (
	"context"

	dashboardAPI "cpusim/dashboard/api/generated"
)

// Manager defines the interface for managing experiments
type Manager interface {
	// Create creates a new experiment
	Create(ctx context.Context, req dashboardAPI.CreateExperimentRequest) (*dashboardAPI.Experiment, error)

	// Get retrieves an experiment by ID
	Get(ctx context.Context, experimentID string) (*dashboardAPI.Experiment, error)

	// List returns a list of experiments with optional filters
	List(ctx context.Context, params dashboardAPI.GetExperimentsParams) (*dashboardAPI.ExperimentListResponse, error)

	// Start starts an experiment (all 4 phases)
	Start(ctx context.Context, experimentID string) (*dashboardAPI.ExperimentOperationResponse, error)

	// Stop stops an experiment (remaining phases)
	Stop(ctx context.Context, experimentID string) (*dashboardAPI.ExperimentOperationResponse, error)

	// GetData retrieves experiment data
	GetData(ctx context.Context, experimentID string, params dashboardAPI.GetExperimentDataParams) (*dashboardAPI.ExperimentDataResponse, error)

	// GetPhases retrieves experiment phase status
	GetPhases(ctx context.Context, experimentID string) (*dashboardAPI.ExperimentPhases, error)
}