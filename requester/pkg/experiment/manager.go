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
	experiments map[string]*Experiment  // Only running experiments
	storage     *storage.FileStorage
	mu          sync.RWMutex
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
		experiments: make(map[string]*Experiment),
		storage:     fileStorage,
	}
}

// StartExperiment starts a new experiment
func (m *Manager) StartExperiment(request generated.StartRequestExperimentRequest) (*generated.RequestExperiment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if experiment already exists in memory (running)
	if _, exists := m.experiments[request.ExperimentId]; exists {
		return nil, fmt.Errorf("experiment already exists and is running")
	}

	// Check if experiment exists in storage (stopped)
	if m.storage.ExperimentExists(request.ExperimentId) {
		return nil, fmt.Errorf("experiment already exists")
	}

	// Create new experiment
	experiment := NewExperiment(request)
	m.experiments[request.ExperimentId] = experiment

	// Start the experiment
	go experiment.Start()

	// Return the experiment status
	return experiment.ToRequestExperiment(), nil
}

// GetExperiment gets an experiment by ID (from memory or storage)
func (m *Manager) GetExperiment(experimentId string) (*generated.RequestExperiment, error) {
	m.mu.RLock()
	// Check if experiment is running (in memory)
	if experiment, exists := m.experiments[experimentId]; exists {
		m.mu.RUnlock()
		return experiment.ToRequestExperiment(), nil
	}
	m.mu.RUnlock()

	// Check if experiment is stored (stopped)
	data, err := m.storage.LoadExperiment(experimentId)
	if err != nil {
		return nil, fmt.Errorf("experiment not found")
	}

	return data.Experiment, nil
}

// StopExperiment stops an experiment
func (m *Manager) StopExperiment(experimentId string) (*generated.StopExperimentResult, error) {
	m.mu.RLock()
	experiment, exists := m.experiments[experimentId]
	m.mu.RUnlock()

	if !exists {
		// Check if it's already stopped and in storage
		_, err := m.storage.LoadExperiment(experimentId)
		if err == nil {
			return nil, fmt.Errorf("experiment already stopped")
		}
		return nil, fmt.Errorf("experiment not found")
	}

	// Stop the experiment
	result, err := experiment.Stop()
	if err != nil {
		return nil, err
	}

	// Save to storage
	experimentData := experiment.ToRequestExperiment()
	stats := experiment.GetStats()
	if err := m.storage.SaveExperiment(experimentData, stats); err != nil {
		// Log error but don't fail the stop operation
		fmt.Printf("Warning: failed to save experiment to storage: %v\n", err)
	}

	// Remove from memory
	m.mu.Lock()
	delete(m.experiments, experimentId)
	m.mu.Unlock()

	return result, nil
}

// GetExperimentStats gets experiment statistics (from memory or storage)
func (m *Manager) GetExperimentStats(experimentId string) (*generated.RequestExperimentStats, error) {
	m.mu.RLock()
	// Check if experiment is running (in memory)
	if experiment, exists := m.experiments[experimentId]; exists {
		m.mu.RUnlock()
		return experiment.GetStats(), nil
	}
	m.mu.RUnlock()

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

	// Add running experiments from memory
	m.mu.RLock()
	for _, experiment := range m.experiments {
		exp := experiment.ToRequestExperiment()

		// Apply status filter if provided
		if statusFilter != nil && *statusFilter != "all" && exp.Status != nil && string(*exp.Status) != *statusFilter {
			continue
		}

		result = append(result, *exp)
	}
	m.mu.RUnlock()

	// Add stopped experiments from storage
	if statusFilter == nil || *statusFilter == "all" || *statusFilter == "stopped" || *statusFilter == "completed" {
		storedExperiments, err := m.storage.ListExperiments()
		if err != nil {
			fmt.Printf("Warning: failed to load stored experiments: %v\n", err)
		} else {
			for _, exp := range storedExperiments {
				// Apply status filter if provided
				if statusFilter != nil && *statusFilter != "all" && exp.Status != nil && string(*exp.Status) != *statusFilter {
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
	var experimentIds []string
	for id := range m.experiments {
		experimentIds = append(experimentIds, id)
	}
	m.mu.RUnlock()

	// Stop each experiment (this will save to storage and clean up memory)
	for _, id := range experimentIds {
		_, err := m.StopExperiment(id)
		if err != nil {
			fmt.Printf("Warning: failed to stop experiment %s: %v\n", id, err)
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