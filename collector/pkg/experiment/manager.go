package experiment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cpusim/collector/pkg/metrics"
	"cpusim/collector/pkg/storage"
)

// Status represents experiment status
type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	StatusTimeout Status = "timeout"
	StatusError   Status = "error"
)

// Experiment represents an active or completed experiment
type Experiment struct {
	ID                  string                  `json:"experimentId"`
	Description         string                  `json:"description,omitempty"`
	StartTime           time.Time               `json:"startTime"`
	EndTime             *time.Time              `json:"endTime,omitempty"`
	Status              Status                  `json:"status"`
	CollectionInterval  time.Duration           `json:"collectionInterval"`
	Timeout             time.Duration           `json:"timeout"`
	IsActive            bool                    `json:"isActive"`
	DataPoints          []metrics.SystemMetrics `json:"dataPoints"`
	DataPointsCollected int                     `json:"dataPointsCollected"`
	LastMetrics         *metrics.SystemMetrics  `json:"lastMetrics,omitempty"`

	// Internal fields
	ctx        context.Context
	cancelFunc context.CancelFunc
	ticker     *time.Ticker
	mu         sync.RWMutex
}

// Manager handles experiment lifecycle
type Manager struct {
	experiments      map[string]*Experiment
	metricsCollector *metrics.Collector
	storage          *storage.FileStorage
	mu               sync.RWMutex
}

// NewManager creates a new experiment manager
func NewManager(metricsCollector *metrics.Collector, storage *storage.FileStorage) *Manager {
	return &Manager{
		experiments:      make(map[string]*Experiment),
		metricsCollector: metricsCollector,
		storage:          storage,
	}
}

// StartExperiment starts a new experiment with the given parameters
func (m *Manager) StartExperiment(id, description string, collectionInterval, timeout time.Duration) (*Experiment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if experiment already exists
	if _, exists := m.experiments[id]; exists {
		return nil, fmt.Errorf("experiment with ID %s already exists", id)
	}

	// Validate experiment ID format (kubernetes-style naming)
	// Pattern: ^[a-z0-9]([a-z0-9-]*[a-z0-9])?$
	// No longer enforcing UUID format to match OpenAPI specification

	// Create experiment context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	experiment := &Experiment{
		ID:                 id,
		Description:        description,
		StartTime:          time.Now(),
		Status:             StatusRunning,
		CollectionInterval: collectionInterval,
		Timeout:            timeout,
		IsActive:           true,
		DataPoints:         make([]metrics.SystemMetrics, 0),
		ctx:                ctx,
		cancelFunc:         cancel,
	}

	// Store experiment
	m.experiments[id] = experiment

	// Start data collection
	go m.collectData(experiment)

	// Start timeout monitor
	go m.monitorTimeout(experiment)

	return experiment, nil
}

// StopExperiment stops an active experiment
func (m *Manager) StopExperiment(id string) (*Experiment, error) {
	m.mu.RLock()
	experiment, exists := m.experiments[id]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("experiment with ID %s not found", id)
	}

	experiment.mu.Lock()
	defer experiment.mu.Unlock()

	if !experiment.IsActive {
		return experiment, nil // Already stopped
	}

	// Stop data collection
	experiment.cancelFunc()
	if experiment.ticker != nil {
		experiment.ticker.Stop()
	}

	// Update experiment status
	now := time.Now()
	experiment.EndTime = &now
	experiment.Status = StatusStopped
	experiment.IsActive = false

	// Save data to storage
	if err := m.storage.SaveExperimentData(experiment.ID, m.convertToStorageFormat(experiment)); err != nil {
		// Log error but don't fail the stop operation
		fmt.Printf("Warning: failed to save experiment data: %v\n", err)
	}

	return experiment, nil
}

// GetExperiment returns experiment information
func (m *Manager) GetExperiment(id string) (*Experiment, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	experiment, exists := m.experiments[id]
	if !exists {
		return nil, fmt.Errorf("experiment with ID %s not found", id)
	}

	experiment.mu.RLock()
	defer experiment.mu.RUnlock()

	// Return a copy to avoid race conditions
	experimentCopy := Experiment{
		ID:                  experiment.ID,
		Description:         experiment.Description,
		StartTime:           experiment.StartTime,
		EndTime:             experiment.EndTime,
		Status:              experiment.Status,
		CollectionInterval:  experiment.CollectionInterval,
		Timeout:             experiment.Timeout,
		IsActive:            experiment.IsActive,
		DataPoints:          make([]metrics.SystemMetrics, len(experiment.DataPoints)),
		DataPointsCollected: experiment.DataPointsCollected,
		LastMetrics:         experiment.LastMetrics,
		// Note: intentionally not copying ctx, cancelFunc, ticker, or mu
	}
	copy(experimentCopy.DataPoints, experiment.DataPoints)

	return &experimentCopy, nil
}

// GetExperimentData returns the collected data for an experiment
func (m *Manager) GetExperimentData(id string) (*storage.ExperimentData, error) {
	m.mu.RLock()
	experiment, exists := m.experiments[id]
	m.mu.RUnlock()

	if !exists {
		// Try to load from storage
		return m.storage.LoadExperimentData(id)
	}

	experiment.mu.RLock()
	defer experiment.mu.RUnlock()

	return m.convertToStorageFormat(experiment), nil
}

// collectData runs the data collection loop for an experiment
func (m *Manager) collectData(experiment *Experiment) {
	experiment.ticker = time.NewTicker(experiment.CollectionInterval)
	defer experiment.ticker.Stop()

	for {
		select {
		case <-experiment.ctx.Done():
			return
		case <-experiment.ticker.C:
			// Collect metrics
			systemMetrics, err := m.metricsCollector.GetCurrentMetrics(experiment.ctx)
			if err != nil {
				fmt.Printf("Error collecting metrics for experiment %s: %v\n", experiment.ID, err)
				continue
			}

			// Store metrics
			experiment.mu.Lock()
			experiment.DataPoints = append(experiment.DataPoints, *systemMetrics)
			experiment.DataPointsCollected = len(experiment.DataPoints)
			experiment.LastMetrics = systemMetrics
			experiment.mu.Unlock()
		}
	}
}

// monitorTimeout handles experiment timeout
func (m *Manager) monitorTimeout(experiment *Experiment) {
	<-experiment.ctx.Done()

	experiment.mu.Lock()
	defer experiment.mu.Unlock()

	// Check if context was cancelled due to timeout or manual stop
	if experiment.ctx.Err() == context.DeadlineExceeded && experiment.IsActive {
		now := time.Now()
		experiment.EndTime = &now
		experiment.Status = StatusTimeout
		experiment.IsActive = false

		// Save data to storage
		if err := m.storage.SaveExperimentData(experiment.ID, m.convertToStorageFormat(experiment)); err != nil {
			fmt.Printf("Warning: failed to save experiment data on timeout: %v\n", err)
		}
	}
}

// convertToStorageFormat converts experiment to storage format
func (m *Manager) convertToStorageFormat(experiment *Experiment) *storage.ExperimentData {
	data := &storage.ExperimentData{
		ExperimentID:       experiment.ID,
		Description:        experiment.Description,
		StartTime:          experiment.StartTime,
		CollectionInterval: int(experiment.CollectionInterval.Milliseconds()),
		Metrics:            make([]storage.MetricDataPoint, 0, len(experiment.DataPoints)),
	}

	if experiment.EndTime != nil {
		data.EndTime = experiment.EndTime
		data.Duration = int(experiment.EndTime.Sub(experiment.StartTime).Seconds())
	}

	// Convert metrics to storage format
	for _, metric := range experiment.DataPoints {
		dataPoint := storage.MetricDataPoint{
			Timestamp: metric.Timestamp,
			SystemMetrics: storage.SystemMetrics{
				CPUUsagePercent:          metric.CPUUsagePercent,
				MemoryUsageBytes:         metric.MemoryUsageBytes,
				MemoryUsagePercent:       metric.MemoryUsagePercent,
				CalculatorServiceHealthy: metric.CalculatorServiceHealthy,
				NetworkIOBytes: storage.NetworkIO{
					BytesReceived:   metric.NetworkIOBytes.BytesReceived,
					BytesSent:       metric.NetworkIOBytes.BytesSent,
					PacketsReceived: metric.NetworkIOBytes.PacketsReceived,
					PacketsSent:     metric.NetworkIOBytes.PacketsSent,
				},
			},
		}
		data.Metrics = append(data.Metrics, dataPoint)
	}

	return data
}

// ListAllExperiments returns summary information for all experiments (active and stored)
func (m *Manager) ListAllExperiments() []ExperimentSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var summaries []ExperimentSummary

	// Add active experiments
	for _, exp := range m.experiments {
		exp.mu.RLock()
		summary := ExperimentSummary{
			ID:                  exp.ID,
			Description:         exp.Description,
			Status:              exp.Status,
			StartTime:           exp.StartTime,
			EndTime:             exp.EndTime,
			IsActive:            exp.IsActive,
			DataPointsCollected: exp.DataPointsCollected,
		}
		if exp.EndTime != nil {
			duration := int(exp.EndTime.Sub(exp.StartTime).Seconds())
			summary.Duration = &duration
		}
		exp.mu.RUnlock()

		summaries = append(summaries, summary)
	}

	return summaries
}

// ExperimentSummary represents a summary of an experiment
type ExperimentSummary struct {
	ID                  string     `json:"experimentId"`
	Description         string     `json:"description,omitempty"`
	Status              Status     `json:"status"`
	StartTime           time.Time  `json:"startTime"`
	EndTime             *time.Time `json:"endTime,omitempty"`
	Duration            *int       `json:"duration,omitempty"` // Duration in seconds
	IsActive            bool       `json:"isActive"`
	DataPointsCollected int        `json:"dataPointsCollected"`
}
