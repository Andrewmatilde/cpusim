package experiment

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	dashboardAPI "cpusim/dashboard/api/generated"
	"cpusim/dashboard/pkg/client"
	"cpusim/dashboard/pkg/target"
)

// ExperimentManager implements the Manager interface
type ExperimentManager struct {
	targetMgr *target.TargetManager
	clientMgr *client.ClientManager
	dataDir   string
	phases    map[string]*dashboardAPI.ExperimentPhases
	phasesMux sync.RWMutex
}

// NewManager creates a new ExperimentManager
func NewManager(targetMgr *target.TargetManager, clientMgr *client.ClientManager, dataDir string) *ExperimentManager {
	if dataDir == "" {
		dataDir = "./data/experiments"
	}

	// Ensure data directory exists
	os.MkdirAll(dataDir, 0755)

	return &ExperimentManager{
		targetMgr: targetMgr,
		clientMgr: clientMgr,
		dataDir:   dataDir,
		phases:    make(map[string]*dashboardAPI.ExperimentPhases),
	}
}

// Create creates a new experiment
func (em *ExperimentManager) Create(ctx context.Context, req dashboardAPI.CreateExperimentRequest) (*dashboardAPI.Experiment, error) {
	// Check if experiment already exists
	expDir := filepath.Join(em.dataDir, req.ExperimentId)
	if _, err := os.Stat(expDir); err == nil {
		return nil, fmt.Errorf("experiment %s already exists", req.ExperimentId)
	}

	// Create experiment
	now := time.Now()
	experiment := &dashboardAPI.Experiment{
		ExperimentId:       req.ExperimentId,
		Description:        req.Description,
		CreatedAt:          now,
		Timeout:            req.Timeout,
		CollectionInterval: req.CollectionInterval,
		TargetHosts:        req.TargetHosts,
		ClientHost:         req.ClientHost,
		RequestConfig:      req.RequestConfig,
	}

	// Save experiment metadata
	if err := em.saveExperiment(req.ExperimentId, experiment); err != nil {
		return nil, fmt.Errorf("failed to save experiment: %w", err)
	}

	return experiment, nil
}

// Get retrieves an experiment by ID
func (em *ExperimentManager) Get(ctx context.Context, experimentID string) (*dashboardAPI.Experiment, error) {
	expFile := filepath.Join(em.dataDir, experimentID, "experiment.json")
	data, err := os.ReadFile(expFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("experiment not found: %s", experimentID)
		}
		return nil, fmt.Errorf("failed to read experiment: %w", err)
	}

	var experiment dashboardAPI.Experiment
	if err := json.Unmarshal(data, &experiment); err != nil {
		return nil, fmt.Errorf("failed to parse experiment: %w", err)
	}

	// Add phase information if available
	em.phasesMux.RLock()
	if phases, ok := em.phases[experimentID]; ok {
		experiment.Phases = *phases
	}
	em.phasesMux.RUnlock()

	return &experiment, nil
}

// List returns a list of experiments with optional filters
func (em *ExperimentManager) List(ctx context.Context, params dashboardAPI.GetExperimentsParams) (*dashboardAPI.ExperimentListResponse, error) {
	entries, err := os.ReadDir(em.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return &dashboardAPI.ExperimentListResponse{
				Experiments: []dashboardAPI.Experiment{},
				Total:       0,
				HasMore:     false,
			}, nil
		}
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	var experiments []dashboardAPI.Experiment
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		exp, err := em.Get(ctx, entry.Name())
		if err != nil {
			// Skip invalid experiments
			continue
		}

		experiments = append(experiments, *exp)
	}

	// Sort by creation time (newest first)
	sort.Slice(experiments, func(i, j int) bool {
		return experiments[i].CreatedAt.After(experiments[j].CreatedAt)
	})

	// Apply limit
	limit := 50
	if params.Limit != 0 {
		limit = params.Limit
	}

	total := len(experiments)
	hasMore := false
	if len(experiments) > limit {
		experiments = experiments[:limit]
		hasMore = true
	}

	return &dashboardAPI.ExperimentListResponse{
		Experiments: experiments,
		Total:       total,
		HasMore:     hasMore,
	}, nil
}

// saveExperiment saves experiment metadata to disk
func (em *ExperimentManager) saveExperiment(experimentID string, experiment *dashboardAPI.Experiment) error {
	expDir := filepath.Join(em.dataDir, experimentID)
	if err := os.MkdirAll(expDir, 0755); err != nil {
		return fmt.Errorf("failed to create experiment directory: %w", err)
	}

	expFile := filepath.Join(expDir, "experiment.json")
	data, err := json.MarshalIndent(experiment, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal experiment: %w", err)
	}

	if err := os.WriteFile(expFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write experiment file: %w", err)
	}

	return nil
}

// GetPhases retrieves experiment phase status
func (em *ExperimentManager) GetPhases(ctx context.Context, experimentID string) (*dashboardAPI.ExperimentPhases, error) {
	em.phasesMux.RLock()
	defer em.phasesMux.RUnlock()

	phases, ok := em.phases[experimentID]
	if !ok {
		return nil, fmt.Errorf("experiment phases not found: %s", experimentID)
	}

	return phases, nil
}

