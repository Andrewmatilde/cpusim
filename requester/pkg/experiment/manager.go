package experiment

import (
	"fmt"
	"sync"
	"time"

	"cpusim/requester/api/generated"
	"cpusim/requester/pkg/storage"
)

// Manager manages all request sending experiments
type Manager struct {
	currentExperiment *Experiment // Current running experiment (nil if no experiment is running)
	storage           *storage.FileStorage
	mu                sync.RWMutex
}

// NewManager creates a new experiment manager
func NewManager() *Manager {
	// Initialize file storage
	fileStorage, err := storage.NewFileStorage("./data/experiments")
	if err != nil {
		// Fallback to temporary directory if default fails
		fileStorage, err = storage.NewFileStorage("/tmp/requester-experiments")
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize experiment storage: %v", err))
		}
	}

	return &Manager{
		currentExperiment: nil,
		storage:           fileStorage,
	}
}

// StartExperiment starts a new experiment
func (m *Manager) StartExperiment(request generated.StartRequestExperimentRequest) (*generated.RequestExperiment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if there is already a running experiment
	if m.currentExperiment != nil {
		currentID := m.currentExperiment.config.ExperimentId

		// If the current experiment has the same ID - return it (idempotent)
		if currentID == request.ExperimentId {
			return m.currentExperiment.ToRequestExperiment(), nil
		}

		// If another experiment is running, reject
		return nil, fmt.Errorf("another experiment %s is already running on this host, please stop it first", currentID)
	}

	// Check if experiment exists in storage (stopped) - cannot restart
	if m.storage.ExperimentExists(request.ExperimentId) {
		return nil, fmt.Errorf("experiment with ID %s already completed, cannot restart", request.ExperimentId)
	}

	// Create new experiment
	experiment := NewExperiment(request, m.storage)
	m.currentExperiment = experiment

	// Start the experiment
	go experiment.Start()

	// Return the experiment status
	return experiment.ToRequestExperiment(), nil
}

// GetExperiment gets an experiment by ID (from memory or storage)
func (m *Manager) GetExperiment(experimentId string) (*generated.RequestExperiment, error) {
	m.mu.RLock()
	experiment := m.currentExperiment
	m.mu.RUnlock()

	// Check if current experiment matches
	if experiment != nil && experiment.config.ExperimentId == experimentId {
		return experiment.ToRequestExperiment(), nil
	}

	// Check if experiment is stored (stopped)
	data, err := m.storage.LoadExperiment(experimentId)
	if err != nil {
		return nil, fmt.Errorf("experiment not found")
	}

	return data.Experiment, nil
}

// StopExperiment stops an experiment
func (m *Manager) StopExperiment(experimentId string) (*generated.StopExperimentResult, error) {
	// Priority 1: Check storage first (source of truth for stopped experiments)
	if data, err := m.storage.LoadExperiment(experimentId); err == nil {
		// Experiment already stopped, return complete data (idempotent)
		var duration int
		if !data.Experiment.EndTime.IsZero() && !data.Experiment.StartTime.IsZero() {
			duration = int(data.Experiment.EndTime.Sub(data.Experiment.StartTime).Seconds())
		}

		return &generated.StopExperimentResult{
			ExperimentId: experimentId,
			EndTime:      data.Experiment.EndTime,
			Duration:     duration,
			FinalStats:   *data.Stats,
		}, nil
	}

	// Priority 2: Check memory for running experiment
	m.mu.RLock()
	experiment := m.currentExperiment
	m.mu.RUnlock()

	if experiment == nil || experiment.config.ExperimentId != experimentId {
		return nil, fmt.Errorf("experiment with ID %s not found", experimentId)
	}

	// Stop the experiment (Stop() will save to storage)
	result, err := experiment.Stop()
	if err != nil {
		return nil, err
	}

	// Clear current experiment
	m.mu.Lock()
	m.currentExperiment = nil
	m.mu.Unlock()

	return result, nil
}

// GetExperimentStats gets experiment statistics (from memory or storage)
func (m *Manager) GetExperimentStats(experimentId string) (*generated.RequestExperimentStats, error) {
	m.mu.RLock()
	experiment := m.currentExperiment
	m.mu.RUnlock()

	// Check if current experiment matches
	if experiment != nil && experiment.config.ExperimentId == experimentId {
		return experiment.GetStats(), nil
	}

	// Check if experiment is stored (stopped)
	data, err := m.storage.LoadExperiment(experimentId)
	if err != nil {
		return nil, fmt.Errorf("experiment not found")
	}

	return data.Stats, nil
}

// ListExperiments lists all experiments with optional status filter (from memory and storage)
func (m *Manager) ListExperiments(statusFilter *string) []generated.RequestExperiment {
	var result []generated.RequestExperiment

	// Add current running experiment if it exists
	m.mu.RLock()
	experiment := m.currentExperiment
	m.mu.RUnlock()

	if experiment != nil {
		exp := experiment.ToRequestExperiment()

		// Apply status filter if provided
		if statusFilter != nil && *statusFilter != "all" && string(exp.Status) != *statusFilter {
			// Don't add, skip to stored experiments
		} else {
			result = append(result, *exp)
		}
	}

	// Add stopped experiments from storage
	if statusFilter == nil || *statusFilter == "all" || *statusFilter == "stopped" || *statusFilter == "completed" {
		storedExperiments, err := m.storage.ListExperiments()
		if err != nil {
			fmt.Printf("Warning: failed to load stored experiments: %v\n", err)
		} else {
			for _, exp := range storedExperiments {
				// Apply status filter if provided
				if statusFilter != nil && *statusFilter != "all" && string(exp.Status) != *statusFilter {
					continue
				}
				result = append(result, *exp)
			}
		}
	}

	return result
}

// StopAllExperiments stops all running experiments
func (m *Manager) StopAllExperiments() {
	m.mu.RLock()
	experiment := m.currentExperiment
	m.mu.RUnlock()

	// Stop current experiment if it exists
	if experiment != nil {
		_, err := m.StopExperiment(experiment.config.ExperimentId)
		if err != nil {
			fmt.Printf("Warning: failed to stop experiment %s: %v\n", experiment.config.ExperimentId, err)
		}
	}
}

// CleanupOldExperiments removes old experiments from storage
func (m *Manager) CleanupOldExperiments(olderThan time.Duration) error {
	// Only running experiments are in memory, so just clean up storage
	return m.storage.CleanupOldExperiments(olderThan)
}

// GetStoragePath returns the storage path for experiments
func (m *Manager) GetStoragePath() string {
	return m.storage.GetStoragePath()
}