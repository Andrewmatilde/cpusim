package experiment

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cpusim/collector/api/generated"
	"cpusim/collector/pkg/metrics"
	"cpusim/collector/pkg/storage"
)

// Experiment represents an active or completed experiment
type Experiment struct {
	ID                  string                            `json:"experimentId"`
	Description         string                            `json:"description,omitempty"`
	StartTime           time.Time                         `json:"startTime"`
	EndTime             *time.Time                        `json:"endTime,omitempty"`
	Status              generated.ExperimentStatusStatus  `json:"status"`
	CollectionInterval  time.Duration                     `json:"collectionInterval"`
	Timeout             time.Duration                     `json:"timeout"`
	IsActive            bool                              `json:"isActive"`
	DataPoints          []metrics.SystemMetrics           `json:"dataPoints"`
	DataPointsCollected int                               `json:"dataPointsCollected"`
	LastMetrics         *metrics.SystemMetrics            `json:"lastMetrics,omitempty"`

	// Internal fields
	storage       *storage.FileStorage
	collector     *metrics.Collector
	ctx           context.Context
	cancelFunc    context.CancelFunc
	ticker        *time.Ticker
	done          chan struct{} // Signals that collection has finished
	mu            sync.RWMutex
}

// Manager handles experiment lifecycle
type Manager struct {
	currentExperiment *Experiment // Current running experiment (nil if no experiment is running)
	metricsCollector  *metrics.Collector
	storage           *storage.FileStorage
	mu                sync.RWMutex
}

// NewManager creates a new experiment manager
func NewManager(metricsCollector *metrics.Collector, storage *storage.FileStorage) *Manager {
	return &Manager{
		currentExperiment: nil,
		metricsCollector:  metricsCollector,
		storage:           storage,
	}
}

// StartExperiment starts a new experiment with the given parameters
func (m *Manager) StartExperiment(id, description string, collectionInterval, timeout time.Duration) (*Experiment, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if there is already a running experiment
	if m.currentExperiment != nil {
		m.currentExperiment.mu.RLock()
		currentID := m.currentExperiment.ID
		currentActive := m.currentExperiment.IsActive
		m.currentExperiment.mu.RUnlock()

		// If the current experiment has the same ID - return it (idempotent)
		if currentID == id {
			return m.currentExperiment, nil
		}

		// If another experiment is running, reject
		if currentActive {
			return nil, fmt.Errorf("another experiment %s is already running on this host, please stop it first", currentID)
		}
	}

	// Check if experiment exists in storage (already completed) - cannot restart
	if m.storage.ExperimentExists(id) {
		return nil, fmt.Errorf("experiment with ID %s already completed, cannot restart", id)
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
		Status:             generated.ExperimentStatusStatusRunning,
		CollectionInterval: collectionInterval,
		Timeout:            timeout,
		IsActive:           true,
		DataPoints:         make([]metrics.SystemMetrics, 0),
		storage:            m.storage,
		collector:          m.metricsCollector,
		ctx:                ctx,
		cancelFunc:         cancel,
		done:               make(chan struct{}),
	}

	// Store as current experiment
	m.currentExperiment = experiment

	// Start data collection
	go experiment.collectData()

	return experiment, nil
}

// StopExperiment stops an active experiment
func (m *Manager) StopExperiment(id string) (*Experiment, error) {
	// Priority 1: Check storage first (source of truth for stopped experiments)
	if data, err := m.storage.LoadExperimentData(id); err == nil {
		// Experiment already stopped, return complete metadata (idempotent)
		// Determine status from data
		status := generated.ExperimentStatusStatusStopped
		if data.EndTime == nil {
			// Data exists but no EndTime, might be error state
			status = generated.ExperimentStatusStatusError
		}

		return &Experiment{
			ID:                  data.ExperimentID,
			Description:         data.Description,
			StartTime:           data.StartTime,
			EndTime:             data.EndTime,
			Status:              status,
			IsActive:            false,
			DataPointsCollected: len(data.Metrics),
		}, nil
	}

	// Priority 2: Check memory for running experiment
	m.mu.RLock()
	experiment := m.currentExperiment
	m.mu.RUnlock()

	if experiment == nil || experiment.ID != id {
		return nil, fmt.Errorf("experiment with ID %s not found", id)
	}

	experiment.mu.RLock()
	isActive := experiment.IsActive
	experiment.mu.RUnlock()

	if !isActive {
		return experiment, nil // Already stopped
	}

	// Trigger stop by cancelling context
	experiment.cancelFunc()

	// Wait for collection to finish (collectData will save to storage)
	<-experiment.done

	// Clear current experiment
	m.mu.Lock()
	m.currentExperiment = nil
	m.mu.Unlock()

	return experiment, nil
}

// GetExperiment returns experiment information
func (m *Manager) GetExperiment(id string) (*Experiment, error) {
	m.mu.RLock()
	experiment := m.currentExperiment
	m.mu.RUnlock()

	// Check if the current experiment matches the requested ID
	if experiment != nil && experiment.ID == id {
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

	// Not in memory, return error
	return nil, fmt.Errorf("experiment with ID %s not found", id)
}

// GetExperimentData returns the collected data for an experiment
func (m *Manager) GetExperimentData(id string) (*storage.ExperimentData, error) {
	m.mu.RLock()
	experiment := m.currentExperiment
	m.mu.RUnlock()

	// Check if current experiment matches
	if experiment != nil && experiment.ID == id {
		return experiment.convertToStorageFormat(), nil
	}

	// Try to load from storage
	return m.storage.LoadExperimentData(id)
}

// collectData runs the data collection loop for an experiment
func (e *Experiment) collectData() {
	e.ticker = time.NewTicker(e.CollectionInterval)
	defer e.ticker.Stop()
	defer close(e.done)

	for {
		select {
		case <-e.ctx.Done():
			// Immediately set end time and status when loop exits
			e.mu.Lock()
			now := time.Now()
			e.EndTime = &now
			e.IsActive = false
			if e.ctx.Err() == context.DeadlineExceeded {
				e.Status = generated.ExperimentStatusStatusTimeout
			} else {
				e.Status = generated.ExperimentStatusStatusStopped
			}
			e.mu.Unlock()

			// Save experiment data to storage
			if err := e.storage.SaveExperimentData(e.ID, e.convertToStorageFormat()); err != nil {
				fmt.Printf("Warning: failed to save experiment data: %v\n", err)
			}

			return

		case <-e.ticker.C:
			// Collect metrics
			systemMetrics, err := e.collector.GetCurrentMetrics(e.ctx)
			if err != nil {
				fmt.Printf("Error collecting metrics for experiment %s: %v\n", e.ID, err)
				continue
			}

			// Store metrics
			e.mu.Lock()
			e.DataPoints = append(e.DataPoints, *systemMetrics)
			e.DataPointsCollected = len(e.DataPoints)
			e.LastMetrics = systemMetrics
			e.mu.Unlock()
		}
	}
}

// convertToStorageFormat converts experiment to storage format
func (e *Experiment) convertToStorageFormat() *storage.ExperimentData {
	e.mu.RLock()
	defer e.mu.RUnlock()

	data := &storage.ExperimentData{
		ExperimentID:       e.ID,
		Description:        e.Description,
		StartTime:          e.StartTime,
		CollectionInterval: int(e.CollectionInterval.Milliseconds()),
		Metrics:            make([]storage.MetricDataPoint, 0, len(e.DataPoints)),
	}

	if e.EndTime != nil {
		data.EndTime = e.EndTime
		data.Duration = int(e.EndTime.Sub(e.StartTime).Seconds())
	}

	// Convert metrics to storage format
	for _, metric := range e.DataPoints {
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
	experiment := m.currentExperiment
	m.mu.RUnlock()

	var summaries []ExperimentSummary

	// Add current experiment if it exists
	if experiment != nil {
		experiment.mu.RLock()
		summary := ExperimentSummary{
			ID:                  experiment.ID,
			Description:         experiment.Description,
			Status:              experiment.Status,
			StartTime:           experiment.StartTime,
			EndTime:             experiment.EndTime,
			IsActive:            experiment.IsActive,
			DataPointsCollected: experiment.DataPointsCollected,
		}
		if experiment.EndTime != nil {
			duration := int(experiment.EndTime.Sub(experiment.StartTime).Seconds())
			summary.Duration = &duration
		}
		experiment.mu.RUnlock()

		summaries = append(summaries, summary)
	}

	return summaries
}

// ExperimentSummary represents a summary of an experiment
type ExperimentSummary struct {
	ID                  string                           `json:"experimentId"`
	Description         string                           `json:"description,omitempty"`
	Status              generated.ExperimentStatusStatus `json:"status"`
	StartTime           time.Time                        `json:"startTime"`
	EndTime             *time.Time                       `json:"endTime,omitempty"`
	Duration            *int                             `json:"duration,omitempty"` // Duration in seconds
	IsActive            bool                             `json:"isActive"`
	DataPointsCollected int                              `json:"dataPointsCollected"`
}
