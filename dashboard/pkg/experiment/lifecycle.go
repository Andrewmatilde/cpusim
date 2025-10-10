package experiment

import (
	"context"
	"fmt"
	"sync"
	"time"

	dashboardAPI "cpusim/dashboard/api/generated"
	"cpusim/dashboard/pkg/client"
)

// Start starts an experiment (all 4 phases)
func (em *ExperimentManager) Start(ctx context.Context, experimentID string) (*dashboardAPI.ExperimentOperationResponse, error) {
	// Get the experiment
	experiment, err := em.Get(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment: %w", err)
	}

	// Initialize experiment phases
	phases := dashboardAPI.ExperimentPhases{
		CollectorStart: dashboardAPI.PhaseStatus{
			Status: dashboardAPI.PhaseStatusStatusPending,
		},
		RequesterStart: dashboardAPI.PhaseStatus{
			Status: dashboardAPI.PhaseStatusStatusPending,
		},
		RequesterStop: dashboardAPI.PhaseStatus{
			Status: dashboardAPI.PhaseStatusStatusPending,
		},
		CollectorStop: dashboardAPI.PhaseStatus{
			Status: dashboardAPI.PhaseStatusStatusPending,
		},
	}

	// Store phases
	em.phasesMux.Lock()
	em.phases[experimentID] = &phases
	em.phasesMux.Unlock()

	// Phase 1: Start collectors on target hosts
	phases.CollectorStart.Status = dashboardAPI.PhaseStatusStatusRunning
	phases.CollectorStart.StartTime = time.Now()

	var failedTargets []string
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Start collectors on all target hosts
	for _, targetHost := range experiment.TargetHosts {
		wg.Add(1)
		go func(th dashboardAPI.HostConfig) {
			defer wg.Done()

			err := em.targetMgr.StartCollector(ctx, th.Name, experimentID, experiment.Description)
			if err != nil {
				mu.Lock()
				failedTargets = append(failedTargets, th.Name)
				mu.Unlock()
				fmt.Printf("Failed to start collector on %s: %v\n", th.Name, err)
			}
		}(targetHost)
	}

	wg.Wait()

	// Update collector start phase
	phases.CollectorStart.EndTime = time.Now()
	if len(failedTargets) > 0 {
		phases.CollectorStart.Status = dashboardAPI.PhaseStatusStatusFailed
		phases.CollectorStart.Message = fmt.Sprintf("Failed to start collectors on: %v", failedTargets)
		return &dashboardAPI.ExperimentOperationResponse{
			ExperimentId: experimentID,
			Message:      "Failed to start experiment",
			Status:       dashboardAPI.ExperimentOperationResponseStatusFailed,
			Phases:       phases,
			Timestamp:    time.Now(),
		}, nil
	}

	phases.CollectorStart.Status = dashboardAPI.PhaseStatusStatusCompleted

	// Phase 2: Start requester on client host
	phases.RequesterStart.Status = dashboardAPI.PhaseStatusStatusRunning
	phases.RequesterStart.StartTime = time.Now()

	// Build target URL (use internal IP if available)
	var targetURL string
	// Find the target host by name
	var targetHost *dashboardAPI.HostConfig
	for i := range experiment.TargetHosts {
		if experiment.TargetHosts[i].Name == experiment.RequestConfig.TargetHostName {
			targetHost = &experiment.TargetHosts[i]
			break
		}
	}

	if targetHost != nil {
		// Prefer internal IP, fallback to external IP
		ip := targetHost.InternalIP
		if ip == "" {
			ip = targetHost.ExternalIP
		}
		targetURL = fmt.Sprintf("http://%s:80", ip)
	}

	if targetURL == "" {
		phases.RequesterStart.Status = dashboardAPI.PhaseStatusStatusFailed
		phases.RequesterStart.Message = "Failed to determine target URL"
		phases.RequesterStart.EndTime = time.Now()
		return &dashboardAPI.ExperimentOperationResponse{
			ExperimentId: experimentID,
			Message:      "Failed to start experiment",
			Status:       dashboardAPI.ExperimentOperationResponseStatusFailed,
			Phases:       phases,
			Timestamp:    time.Now(),
		}, nil
	}

	// Determine duration (default to 60 seconds if not specified)
	duration := 60
	if experiment.Timeout != 0 {
		duration = experiment.Timeout
	}

	// Start requester
	requesterConfig := client.RequesterConfig{
		ExperimentID:   experimentID,
		TargetURL:      targetURL,
		QPS:            experiment.RequestConfig.Qps,
		Duration:       duration,
		RequestTimeout: 5000, // Default 5s timeout
	}

	err = em.clientMgr.StartRequester(ctx, experiment.ClientHost.Name, requesterConfig)
	phases.RequesterStart.EndTime = time.Now()

	if err != nil {
		phases.RequesterStart.Status = dashboardAPI.PhaseStatusStatusFailed
		phases.RequesterStart.Message = fmt.Sprintf("Failed to start requester: %v", err)
		return &dashboardAPI.ExperimentOperationResponse{
			ExperimentId: experimentID,
			Message:      "Failed to start experiment",
			Status:       dashboardAPI.ExperimentOperationResponseStatusFailed,
			Phases:       phases,
			Timestamp:    time.Now(),
		}, nil
	}

	phases.RequesterStart.Status = dashboardAPI.PhaseStatusStatusCompleted
	phases.RequesterStart.Message = "Requester started successfully"

	// Return success (phases 3 and 4 will be handled by Stop)
	return &dashboardAPI.ExperimentOperationResponse{
		ExperimentId: experimentID,
		Message:      "Experiment started successfully",
		Status:       dashboardAPI.ExperimentOperationResponseStatusSuccess,
		Phases:       phases,
		Timestamp:    time.Now(),
	}, nil
}

// Stop stops an experiment (remaining phases)
func (em *ExperimentManager) Stop(ctx context.Context, experimentID string) (*dashboardAPI.ExperimentOperationResponse, error) {
	// Get the experiment
	experiment, err := em.Get(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment: %w", err)
	}

	// Get existing phases
	em.phasesMux.RLock()
	phases, ok := em.phases[experimentID]
	em.phasesMux.RUnlock()

	if !ok {
		// Create default phases if not exist
		phases = &dashboardAPI.ExperimentPhases{
			CollectorStart: dashboardAPI.PhaseStatus{Status: dashboardAPI.PhaseStatusStatusPending},
			RequesterStart: dashboardAPI.PhaseStatus{Status: dashboardAPI.PhaseStatusStatusPending},
			RequesterStop:  dashboardAPI.PhaseStatus{Status: dashboardAPI.PhaseStatusStatusPending},
			CollectorStop:  dashboardAPI.PhaseStatus{Status: dashboardAPI.PhaseStatusStatusPending},
		}
		em.phasesMux.Lock()
		em.phases[experimentID] = phases
		em.phasesMux.Unlock()
	}

	// Phase 3: Stop requester
	phases.RequesterStop.Status = dashboardAPI.PhaseStatusStatusRunning
	phases.RequesterStop.StartTime = time.Now()

	_, err = em.clientMgr.StopRequester(ctx, experiment.ClientHost.Name, experimentID)
	phases.RequesterStop.EndTime = time.Now()

	if err != nil {
		phases.RequesterStop.Status = dashboardAPI.PhaseStatusStatusFailed
		phases.RequesterStop.Message = fmt.Sprintf("Failed to stop requester: %v", err)
	} else {
		phases.RequesterStop.Status = dashboardAPI.PhaseStatusStatusCompleted
	}

	// Phase 4: Stop collectors and collect data
	phases.CollectorStop.Status = dashboardAPI.PhaseStatusStatusRunning
	phases.CollectorStop.StartTime = time.Now()

	var failedTargets []string
	var wg sync.WaitGroup
	var mu sync.Mutex
	collectedData := make(map[string]interface{})

	for _, targetHost := range experiment.TargetHosts {
		wg.Add(1)
		go func(th dashboardAPI.HostConfig) {
			defer wg.Done()

			data, err := em.targetMgr.StopCollector(ctx, th.Name, experimentID)
			if err != nil {
				mu.Lock()
				failedTargets = append(failedTargets, th.Name)
				mu.Unlock()
				fmt.Printf("Failed to stop collector on %s: %v\n", th.Name, err)
			} else {
				mu.Lock()
				collectedData[th.Name] = data
				mu.Unlock()
			}
		}(targetHost)
	}

	wg.Wait()

	phases.CollectorStop.EndTime = time.Now()

	if len(failedTargets) > 0 {
		phases.CollectorStop.Status = dashboardAPI.PhaseStatusStatusFailed
		phases.CollectorStop.Message = fmt.Sprintf("Failed to stop collectors on: %v", failedTargets)
	} else {
		phases.CollectorStop.Status = dashboardAPI.PhaseStatusStatusCompleted
	}

	// Determine overall status
	var overallStatus dashboardAPI.ExperimentOperationResponseStatus
	if len(failedTargets) > 0 || (phases.RequesterStop.Status == dashboardAPI.PhaseStatusStatusFailed) {
		overallStatus = dashboardAPI.ExperimentOperationResponseStatusPartial
	} else {
		overallStatus = dashboardAPI.ExperimentOperationResponseStatusSuccess
	}

	return &dashboardAPI.ExperimentOperationResponse{
		ExperimentId: experimentID,
		Message:      "Experiment stopped",
		Status:       overallStatus,
		Phases:       *phases,
		Timestamp:    time.Now(),
	}, nil
}

