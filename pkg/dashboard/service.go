package dashboard

import (
	"context"
	collectorAPI "cpusim/collector/api/generated"
	requesterAPI "cpusim/requester/api/generated"
	"fmt"
	"time"

	"cpusim/pkg/exp"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Service manages dashboard experiments using the exp framework
type Service struct {
	exp.Manager[*ExperimentData]

	fs           exp.FileStorage[*ExperimentData]
	groupStorage *GroupStorage
	logger       zerolog.Logger
	config       Config

	// HTTP clients for sub-experiments
	collectorClients map[string]CollectorClient // key: host name
	requesterClient  RequesterClient
}

// CollectorClient interface for communicating with collector services
type CollectorClient interface {
	StartExperiment(ctx context.Context, experimentID string, timeout time.Duration) error
	StopExperiment(ctx context.Context, experimentID string) error
	GetExperiment(ctx context.Context, experimentID string) (*collectorAPI.ExperimentData, error)
	GetStatus(ctx context.Context) (string, string, error) // returns status, currentExperimentID, error
}

// RequesterClient interface for communicating with requester services
type RequesterClient interface {
	StartExperiment(ctx context.Context, experimentID string, timeout time.Duration, qps int) error
	StopExperiment(ctx context.Context, experimentID string) error
	GetExperiment(ctx context.Context, experimentID string) (*requesterAPI.RequestExperimentStats, error)
	GetStatus(ctx context.Context) (string, string, error) // returns status, currentExperimentID, error
}

// NewService creates a new dashboard service
func NewService(storagePath string, config Config, logger zerolog.Logger) (*Service, error) {
	fs, err := exp.NewFileStorage[*ExperimentData](storagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file storage: %w", err)
	}

	// Create group storage (in a subdirectory)
	groupStoragePath := storagePath + "/groups"
	groupStorage, err := NewGroupStorage(groupStoragePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create group storage: %w", err)
	}

	s := &Service{
		fs:               *fs,
		groupStorage:     groupStorage,
		logger:           logger,
		config:           config,
		collectorClients: make(map[string]CollectorClient),
	}

	// Create collector function
	collectFunc := func(ctx context.Context, params gin.Params) (*ExperimentData, error) {
		experimentID := ""
		qps := 0

		for _, param := range params {
			if param.Key == "experimentID" {
				experimentID = param.Value
			} else if param.Key == "qps" {
				fmt.Sscanf(param.Value, "%d", &qps)
			}
		}

		return s.runExperiment(ctx, experimentID, qps)
	}

	// Create and embed the manager
	s.Manager = *exp.NewManager[*ExperimentData](*fs, collectFunc, logger)

	return s, nil
}

// SetCollectorClient sets the collector client for a specific host
func (s *Service) SetCollectorClient(hostName string, client CollectorClient) {
	s.collectorClients[hostName] = client
}

// SetRequesterClient sets the requester client
func (s *Service) SetRequesterClient(client RequesterClient) {
	s.requesterClient = client
}

// StartExperiment starts a new dashboard experiment
func (s *Service) StartExperiment(id string, timeout time.Duration, qps int) error {
	// Check status before starting
	status := s.GetStatus()
	if status != exp.Pending {
		return fmt.Errorf("cannot start experiment: current status is %s, must be %s", status, exp.Pending)
	}

	s.logger.Info().
		Str("experiment_id", id).
		Int("num_targets", len(s.config.TargetHosts)).
		Int("qps", qps).
		Msg("Starting dashboard experiment")

	// Pass experiment ID and QPS through params
	params := gin.Params{
		{Key: "experimentID", Value: id},
		{Key: "qps", Value: fmt.Sprintf("%d", qps)},
	}
	return s.Manager.Start(id, timeout, params)
}

// StopExperiment stops the current running experiment
func (s *Service) StopExperiment() error {
	status := s.GetStatus()
	if status != exp.Running {
		return fmt.Errorf("cannot stop experiment: current status is %s, must be %s", status, exp.Running)
	}

	return s.Manager.Stop()
}

// StopAll stops all sub-experiments and cleans up state
func (s *Service) StopAll(experimentID string) error {
	s.logger.Warn().
		Str("experiment_id", experimentID).
		Msg("Stopping all sub-experiments (cleanup)")

	// Use a fresh context for cleanup operations since the experiment context may be cancelled
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cleanupCancel()

	var errors []ExperimentError

	// Stop requester first
	if s.requesterClient != nil {
		if err := s.requesterClient.StopExperiment(cleanupCtx, experimentID); err != nil {
			s.logger.Error().Err(err).Msg("Failed to stop requester")
			errors = append(errors, ExperimentError{
				Timestamp: time.Now(),
				Phase:     "stop_requester",
				Message:   err.Error(),
			})
		}
	}

	// Stop all collectors
	for hostName, client := range s.collectorClients {
		if err := client.StopExperiment(cleanupCtx, experimentID); err != nil {
			s.logger.Error().
				Err(err).
				Str("host", hostName).
				Msg("Failed to stop collector")
			errors = append(errors, ExperimentError{
				Timestamp: time.Now(),
				Phase:     "stop_collector",
				HostName:  hostName,
				Message:   err.Error(),
			})
		}
	}

	// Stop the main experiment if running
	if s.GetStatus() == exp.Running {
		if err := s.StopExperiment(); err != nil {
			s.logger.Error().Err(err).Msg("Failed to stop main experiment")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("stopAll completed with %d errors", len(errors))
	}

	s.logger.Info().Msg("StopAll completed successfully")
	return nil
}

// GetExperiment retrieves experiment data by ID
func (s *Service) GetExperiment(id string) (*ExperimentData, error) {
	return s.fs.Load(id)
}

// ListExperiments lists all stored experiments
func (s *Service) ListExperiments() ([]exp.ExperimentInfo, error) {
	return s.Manager.ListExperiments()
}

// HostStatus represents the status of a host
type HostStatus struct {
	Name                string
	Status              string
	CurrentExperimentID string
	Error               string
}

// GetHostsStatus queries the status of all target and client hosts
func (s *Service) GetHostsStatus(ctx context.Context) ([]HostStatus, *HostStatus, error) {
	targetHostsStatus := make([]HostStatus, 0, len(s.config.TargetHosts))

	// Query all target hosts (collectors)
	for _, target := range s.config.TargetHosts {
		client, ok := s.collectorClients[target.Name]
		if !ok {
			targetHostsStatus = append(targetHostsStatus, HostStatus{
				Name:   target.Name,
				Status: "Error",
				Error:  "collector client not configured",
			})
			continue
		}

		status, expID, err := client.GetStatus(ctx)
		if err != nil {
			targetHostsStatus = append(targetHostsStatus, HostStatus{
				Name:   target.Name,
				Status: "Error",
				Error:  err.Error(),
			})
		} else {
			targetHostsStatus = append(targetHostsStatus, HostStatus{
				Name:                target.Name,
				Status:              status,
				CurrentExperimentID: expID,
			})
		}
	}

	// Query client host (requester)
	var clientHostStatus *HostStatus
	if s.requesterClient == nil {
		clientHostStatus = &HostStatus{
			Name:   s.config.ClientHost.Name,
			Status: "Error",
			Error:  "requester client not configured",
		}
	} else {
		status, expID, err := s.requesterClient.GetStatus(ctx)
		if err != nil {
			clientHostStatus = &HostStatus{
				Name:   s.config.ClientHost.Name,
				Status: "Error",
				Error:  err.Error(),
			}
		} else {
			clientHostStatus = &HostStatus{
				Name:                s.config.ClientHost.Name,
				Status:              status,
				CurrentExperimentID: expID,
			}
		}
	}

	return targetHostsStatus, clientHostStatus, nil
}

// runExperiment executes the complete dashboard experiment
func (s *Service) runExperiment(ctx context.Context, experimentID string, qps int) (*ExperimentData, error) {
	data := &ExperimentData{
		Config:           s.config,
		StartTime:        time.Now(),
		Status:           "running",
		CollectorResults: make(map[string]CollectorResult),
		Errors:           make([]ExperimentError, 0),
	}

	// Phase 1: Start collectors on all target hosts
	s.logger.Info().Msg("Phase 1: Starting collectors on all targets")
	for _, target := range s.config.TargetHosts {
		client, ok := s.collectorClients[target.Name]
		if !ok {
			err := fmt.Errorf("collector client not found for host: %s", target.Name)
			s.logger.Error().Err(err).Str("host", target.Name).Msg("Collector client missing")
			data.Errors = append(data.Errors, ExperimentError{
				Timestamp: time.Now(),
				Phase:     "collector_start",
				HostName:  target.Name,
				Message:   err.Error(),
			})
			data.CollectorResults[target.Name] = CollectorResult{
				HostName: target.Name,
				Status:   "failed",
				Error:    err.Error(),
			}
			// Rollback: stop all
			s.StopAll(experimentID)
			return data, err
		}

		// Start collector experiment
		// Use the experiment timeout from the context or a default value
		timeout := 60 * time.Second
		if deadline, ok := ctx.Deadline(); ok {
			timeout = time.Until(deadline)
		}
		if err := client.StartExperiment(ctx, experimentID, timeout); err != nil {
			s.logger.Error().
				Err(err).
				Str("host", target.Name).
				Msg("Failed to start collector")
			data.Errors = append(data.Errors, ExperimentError{
				Timestamp: time.Now(),
				Phase:     "collector_start",
				HostName:  target.Name,
				Message:   err.Error(),
			})
			data.CollectorResults[target.Name] = CollectorResult{
				HostName: target.Name,
				Status:   "failed",
				Error:    err.Error(),
			}
			// Rollback: stop all
			s.StopAll(experimentID)
			return data, err
		}

		data.CollectorResults[target.Name] = CollectorResult{
			HostName: target.Name,
			Status:   "started",
		}
		s.logger.Info().Str("host", target.Name).Msg("Collector started successfully")
	}

	// Phase 2: Start requester on client host
	s.logger.Info().Msg("Phase 2: Starting requester on client")
	if s.requesterClient == nil {
		err := fmt.Errorf("requester client not configured")
		s.logger.Error().Err(err).Msg("Requester client missing")
		data.Errors = append(data.Errors, ExperimentError{
			Timestamp: time.Now(),
			Phase:     "requester_start",
			Message:   err.Error(),
		})
		data.RequesterResult = &RequesterResult{
			Status: "failed",
			Error:  err.Error(),
		}
		// Rollback: stop all
		s.StopAll(experimentID)
		return data, err
	}

	// Use the experiment timeout from the context or a default value
	timeout := 60 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}
	if err := s.requesterClient.StartExperiment(ctx, experimentID, timeout, qps); err != nil {
		s.logger.Error().Err(err).Msg("Failed to start requester")
		data.Errors = append(data.Errors, ExperimentError{
			Timestamp: time.Now(),
			Phase:     "requester_start",
			Message:   err.Error(),
		})
		data.RequesterResult = &RequesterResult{
			Status: "failed",
			Error:  err.Error(),
		}
		// Rollback: stop all
		s.StopAll(experimentID)
		return data, err
	}

	data.RequesterResult = &RequesterResult{
		Status: "started",
	}
	s.logger.Info().Msg("Requester started successfully")

	// Wait for completion or cancellation
	<-ctx.Done()

	// Phase 3: Stop all sub-experiments
	s.logger.Info().Msg("Phase 3: Stopping all sub-experiments")
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer stopCancel()

	// Stop all collectors
	for hostName := range data.CollectorResults {
		client := s.collectorClients[hostName]
		if err := client.StopExperiment(stopCtx, experimentID); err != nil {
			s.logger.Warn().Err(err).Str("host", hostName).Msg("Failed to stop collector")
		} else {
			s.logger.Info().Str("host", hostName).Msg("Collector stopped successfully")
		}
	}

	// Stop requester
	if err := s.requesterClient.StopExperiment(stopCtx, experimentID); err != nil {
		s.logger.Warn().Err(err).Msg("Failed to stop requester")
	} else {
		s.logger.Info().Msg("Requester stopped successfully")
	}

	// Phase 4: Collect results
	// Use a fresh context for collection since the experiment context is cancelled
	collectCtx := context.Background()
	collectCtx, collectCancel := context.WithTimeout(collectCtx, 10*time.Second)
	defer collectCancel()

	s.logger.Info().Msg("Phase 4: Collecting results from sub-experiments")
	data.EndTime = time.Now()
	data.Duration = data.EndTime.Sub(data.StartTime).Seconds()

	// Collect collector results
	for hostName := range data.CollectorResults {
		client := s.collectorClients[hostName]
		if collectorData, err := client.GetExperiment(collectCtx, experimentID); err == nil {
			data.CollectorResults[hostName] = CollectorResult{
				HostName: hostName,
				Status:   "completed",
				Data:     collectorData,
			}
		} else {
			s.logger.Error().Err(err).Str("host", hostName).Msg("Failed to get collector results")
			result := data.CollectorResults[hostName]
			result.Status = "failed"
			result.Error = err.Error()
			data.CollectorResults[hostName] = result
		}
	}

	// Collect requester results
	if requesterStats, err := s.requesterClient.GetExperiment(collectCtx, experimentID); err == nil {
		data.RequesterResult = &RequesterResult{
			Status: "completed",
			Stats:  requesterStats,
		}
	} else {
		s.logger.Error().Err(err).Msg("Failed to get requester results")
		data.RequesterResult.Status = "failed"
		data.RequesterResult.Error = err.Error()
	}

	// Determine overall status
	if len(data.Errors) > 0 {
		data.Status = "failed"
	} else {
		data.Status = "completed"
	}

	s.logger.Info().
		Str("status", data.Status).
		Float64("duration", data.Duration).
		Msg("Dashboard experiment completed")

	return data, nil
}

// StartExperimentGroup starts a new experiment group with QPS range testing
// Supports resume: if the group already exists and is "running" or "failed", it will continue from where it left off
func (s *Service) StartExperimentGroup(groupID string, description string, config ExperimentGroupConfig) error {
	// Check if service is idle
	status := s.GetStatus()
	if status != exp.Pending {
		return fmt.Errorf("cannot start experiment group: service is %s, must be Pending", status)
	}

	// Generate QPS values from range
	qpsValues := make([]int, 0)
	for qps := config.QPSMin; qps <= config.QPSMax; qps += config.QPSStep {
		qpsValues = append(qpsValues, qps)
	}
	if len(qpsValues) == 0 {
		return fmt.Errorf("invalid QPS range: min=%d, max=%d, step=%d produces no values", config.QPSMin, config.QPSMax, config.QPSStep)
	}

	// Try to load existing group (for resume functionality)
	existingGroup, err := s.groupStorage.Load(groupID)
	var group *ExperimentGroup

	if err == nil {
		// Group exists, check if we can resume
		if existingGroup.Status == "completed" {
			return fmt.Errorf("experiment group %s already completed", groupID)
		}

		s.logger.Info().
			Str("group_id", groupID).
			Int("completed_qps_points", len(existingGroup.QPSPoints)).
			Int("total_qps_points", len(qpsValues)).
			Msg("Resuming existing experiment group")

		group = existingGroup
		// Update config in case it changed
		group.Config = config
		group.Status = "running"
	} else {
		// Create new experiment group
		s.logger.Info().
			Str("group_id", groupID).
			Int("qps_points", len(qpsValues)).
			Int("repeat_per_qps", config.RepeatCount).
			Int("timeout", config.Timeout).
			Msg("Starting new experiment group")

		// Initialize QPSPoints
		qpsPoints := make([]QPSPoint, 0, len(qpsValues))
		for _, qps := range qpsValues {
			qpsPoints = append(qpsPoints, QPSPoint{
				QPS:         qps,
				Experiments: make([]string, 0, config.RepeatCount),
				Statistics:  nil,
				Status:      "pending",
			})
		}

		group = &ExperimentGroup{
			GroupID:     groupID,
			Description: description,
			Config:      config,
			QPSPoints:   qpsPoints,
			StartTime:   time.Now(),
			Status:      "running",
			CurrentQPS:  0,
			CurrentRun:  0,
		}
	}

	// Save initial/resumed group state
	if err := s.groupStorage.Save(groupID, group); err != nil {
		return fmt.Errorf("failed to save experiment group: %w", err)
	}

	// Execute the experiment group
	return s.executeExperimentGroup(groupID, group)
}

// executeExperimentGroup runs the experiments for a group (common logic for both start and resume)
func (s *Service) executeExperimentGroup(groupID string, group *ExperimentGroup) error {
	config := group.Config

	// Run experiments for each QPS value
	for qpsIdx, qpsPoint := range group.QPSPoints {
		qps := qpsPoint.QPS
		group.CurrentQPS = qps

		// Skip completed QPS points (for resume)
		if qpsPoint.Status == "completed" {
			s.logger.Info().
				Str("group_id", groupID).
				Int("qps", qps).
				Int("completed_runs", len(qpsPoint.Experiments)).
				Msg("Skipping completed QPS point")
			continue
		}

		s.logger.Info().
			Str("group_id", groupID).
			Int("qps", qps).
			Int("qps_idx", qpsIdx+1).
			Int("total_qps", len(group.QPSPoints)).
			Msg("Starting QPS point experiments")

		// Update QPS point status
		group.QPSPoints[qpsIdx].Status = "running"
		if err := s.groupStorage.Save(groupID, group); err != nil {
			s.logger.Error().Err(err).Msg("Failed to update group status")
		}

		// Determine starting run (for resume)
		startRun := 1
		if len(qpsPoint.Experiments) > 0 {
			startRun = len(qpsPoint.Experiments) + 1
		}

		// Run RepeatCount experiments for this QPS
		for run := startRun; run <= config.RepeatCount; run++ {
			group.CurrentRun = run
			if err := s.groupStorage.Save(groupID, group); err != nil {
				s.logger.Error().Err(err).Msg("Failed to update group status")
			}

			// Generate experiment ID
			expID := fmt.Sprintf("%s-qps-%d-run-%d", groupID, qps, run)

			s.logger.Info().
				Str("group_id", groupID).
				Int("qps", qps).
				Int("run", run).
				Int("total_runs", config.RepeatCount).
				Str("experiment_id", expID).
				Msg("Starting experiment")

			// Add experiment to QPS point
			group.QPSPoints[qpsIdx].Experiments = append(group.QPSPoints[qpsIdx].Experiments, expID)

			// Start single experiment
			timeout := time.Duration(config.Timeout) * time.Second
			err := s.StartExperiment(expID, timeout, qps)
			if err != nil {
				s.logger.Error().
					Err(err).
					Str("experiment_id", expID).
					Msg("Failed to start experiment")

				group.Status = "failed"
				group.QPSPoints[qpsIdx].Status = "failed"
				group.EndTime = time.Now()
				if saveErr := s.groupStorage.Save(groupID, group); saveErr != nil {
					s.logger.Error().Err(saveErr).Msg("Failed to save failed group state")
				}
				return fmt.Errorf("failed to start experiment %s: %w", expID, err)
			}

			// Wait for experiment to complete
			s.logger.Info().Str("experiment_id", expID).Msg("Waiting for experiment to complete")
			for s.GetStatus() == exp.Running {
				time.Sleep(1 * time.Second)
			}

			s.logger.Info().
				Str("experiment_id", expID).
				Int("qps", qps).
				Int("run", run).
				Msg("Experiment completed")

			// Optional delay between experiments
			if run < config.RepeatCount && config.DelayBetween > 0 {
				s.logger.Info().
					Int("delay_seconds", config.DelayBetween).
					Msg("Waiting before next experiment")
				time.Sleep(time.Duration(config.DelayBetween) * time.Second)
			}

			// Save updated group state
			if err := s.groupStorage.Save(groupID, group); err != nil {
				s.logger.Error().Err(err).Msg("Failed to save group state")
			}
		}

		// Calculate statistics for this QPS point
		s.logger.Info().
			Str("group_id", groupID).
			Int("qps", qps).
			Msg("Calculating statistics for QPS point")

		experiments := make([]*ExperimentData, 0, len(group.QPSPoints[qpsIdx].Experiments))
		for _, expID := range group.QPSPoints[qpsIdx].Experiments {
			expData, err := s.GetExperiment(expID)
			if err != nil {
				s.logger.Warn().
					Err(err).
					Str("experiment_id", expID).
					Msg("Failed to load experiment data for statistics")
				continue
			}
			experiments = append(experiments, expData)
		}

		if len(experiments) > 0 {
			group.QPSPoints[qpsIdx].Statistics = s.calculateSteadyStateStats(experiments)
		}
		group.QPSPoints[qpsIdx].Status = "completed"

		// Save updated group with statistics
		if err := s.groupStorage.Save(groupID, group); err != nil {
			s.logger.Error().Err(err).Msg("Failed to save group state with statistics")
		}

		s.logger.Info().
			Str("group_id", groupID).
			Int("qps", qps).
			Int("completed_runs", len(group.QPSPoints[qpsIdx].Experiments)).
			Msg("QPS point completed")
	}

	// Mark group as completed
	group.Status = "completed"
	group.EndTime = time.Now()
	if err := s.groupStorage.Save(groupID, group); err != nil {
		s.logger.Error().Err(err).Msg("Failed to save final group state")
		return err
	}

	s.logger.Info().
		Str("group_id", groupID).
		Int("qps_points", len(group.QPSPoints)).
		Msg("Experiment group completed successfully")

	return nil
}

// ResumeExperimentGroup resumes an incomplete experiment group
func (s *Service) ResumeExperimentGroup(groupID string) error {
	// Check if service is idle
	status := s.GetStatus()
	if status != exp.Pending {
		return fmt.Errorf("cannot resume experiment group: service is %s, must be Pending", status)
	}

	// Load existing group
	group, err := s.groupStorage.Load(groupID)
	if err != nil {
		return fmt.Errorf("failed to load experiment group: %w", err)
	}

	// Check if group is already completed
	if group.Status == "completed" {
		return fmt.Errorf("experiment group %s already completed", groupID)
	}

	s.logger.Info().
		Str("group_id", groupID).
		Str("status", group.Status).
		Int("qps_points", len(group.QPSPoints)).
		Msg("Resuming experiment group")

	// Update status and continue execution
	group.Status = "running"
	if err := s.groupStorage.Save(groupID, group); err != nil {
		return fmt.Errorf("failed to save experiment group: %w", err)
	}

	// Execute the experiment group (same logic as StartExperimentGroup)
	return s.executeExperimentGroup(groupID, group)
}

// GetExperimentGroup retrieves an experiment group by ID
func (s *Service) GetExperimentGroup(groupID string) (*ExperimentGroup, error) {
	return s.groupStorage.Load(groupID)
}

// ListExperimentGroups lists all experiment groups
// Statistics are already calculated and saved per QPS point during group execution
func (s *Service) ListExperimentGroups() ([]*ExperimentGroup, error) {
	groups, err := s.groupStorage.List()
	if err != nil {
		return nil, err
	}

	return groups, nil
}

// GetExperimentGroupWithDetails retrieves an experiment group with all experiment details
// Statistics are already calculated and saved per QPS point during group execution
func (s *Service) GetExperimentGroupWithDetails(groupID string) (*ExperimentGroup, []*ExperimentData, error) {
	group, err := s.groupStorage.Load(groupID)
	if err != nil {
		return nil, nil, err
	}

	// Collect all experiments from all QPS points
	experiments := make([]*ExperimentData, 0)
	for _, qpsPoint := range group.QPSPoints {
		for _, expID := range qpsPoint.Experiments {
			expData, err := s.GetExperiment(expID)
			if err != nil {
				s.logger.Warn().
					Err(err).
					Str("experiment_id", expID).
					Msg("Failed to load experiment data")
				continue
			}
			experiments = append(experiments, expData)
		}
	}

	return group, experiments, nil
}

// calculateSteadyStateStats calculates steady-state statistics with confidence intervals
// for each host across all experiments in the group
func (s *Service) calculateSteadyStateStats(experiments []*ExperimentData) map[string]*SteadyStateStats {
	if len(experiments) == 0 {
		s.logger.Warn().Msg("calculateSteadyStateStats: no experiments")
		return nil
	}

	stats := make(map[string]*SteadyStateStats)

	// Group metrics by host
	hostMetrics := make(map[string][]float64) // key: host name, value: steady-state mean CPU for each experiment

	for expIdx, exp := range experiments {
		if exp.CollectorResults == nil {
			s.logger.Warn().Int("exp_idx", expIdx).Msg("Experiment has nil CollectorResults")
			continue
		}

		for hostName, result := range exp.CollectorResults {
			if result.Data == nil {
				s.logger.Warn().
					Int("exp_idx", expIdx).
					Str("host", hostName).
					Msg("Collector result has nil Data")
				continue
			}
			if result.Data.Metrics == nil {
				s.logger.Warn().
					Int("exp_idx", expIdx).
					Str("host", hostName).
					Msg("Collector data has nil Metrics")
				continue
			}
			if len(result.Data.Metrics) == 0 {
				s.logger.Warn().
					Int("exp_idx", expIdx).
					Str("host", hostName).
					Msg("Collector data has empty Metrics")
				continue
			}

			// Calculate steady-state mean for this experiment (last 90% of data)
			metrics := result.Data.Metrics
			steadyStateStart := len(metrics) / 10 // Skip first 10%
			if steadyStateStart >= len(metrics) {
				steadyStateStart = 0
			}

			var cpuSum float64
			cpuCount := 0
			for i := steadyStateStart; i < len(metrics); i++ {
				cpuSum += float64(metrics[i].SystemMetrics.CpuUsagePercent)
				cpuCount++
			}

			if cpuCount > 0 {
				steadyStateMean := cpuSum / float64(cpuCount)
				s.logger.Debug().
					Int("exp_idx", expIdx).
					Str("host", hostName).
					Int("metric_count", len(metrics)).
					Float64("steady_state_mean", steadyStateMean).
					Msg("Calculated steady-state mean for experiment")
				hostMetrics[hostName] = append(hostMetrics[hostName], steadyStateMean)
			}
		}
	}

	s.logger.Info().Int("host_count", len(hostMetrics)).Msg("Grouped metrics by host")

	// Calculate statistics for each host
	for hostName, cpuValues := range hostMetrics {
		if len(cpuValues) == 0 {
			continue
		}

		s.logger.Info().
			Str("host", hostName).
			Int("sample_size", len(cpuValues)).
			Msg("Calculating confidence interval")
		stats[hostName] = calculateConfidenceInterval(cpuValues, 0.95)
	}

	s.logger.Info().Int("stats_count", len(stats)).Msg("Calculated steady-state statistics")

	return stats
}

// calculateConfidenceInterval calculates statistics and confidence interval for a set of values
func calculateConfidenceInterval(values []float64, confidenceLevel float64) *SteadyStateStats {
	n := len(values)
	if n == 0 {
		return nil
	}

	// Calculate mean
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(n)

	// Calculate standard deviation
	var varianceSum float64
	for _, v := range values {
		diff := v - mean
		varianceSum += diff * diff
	}
	variance := varianceSum / float64(n-1) // Sample variance (n-1)
	stdDev := 0.0
	if variance > 0 {
		stdDev = sqrt(variance)
	}

	// Calculate standard error
	se := stdDev / sqrt(float64(n))

	// t-values for 95% confidence interval (two-tailed)
	// Map of degrees of freedom (n-1) to t-value
	tValues := map[int]float64{
		1: 12.706, 2: 4.303, 3: 3.182, 4: 2.776, 5: 2.571,
		6: 2.447, 7: 2.365, 8: 2.306, 9: 2.262, 10: 2.228,
		11: 2.201, 12: 2.179, 13: 2.160, 14: 2.145, 15: 2.131,
		16: 2.120, 17: 2.110, 18: 2.101, 19: 2.093, 20: 2.086,
		25: 2.060, 30: 2.042, 40: 2.021, 50: 2.009, 60: 2.000,
		80: 1.990, 100: 1.984, 120: 1.980,
	}

	// Get appropriate t-value
	df := n - 1
	tValue := 1.96 // Default to z-value for large samples

	if df <= 20 {
		if val, ok := tValues[df]; ok {
			tValue = val
		}
	} else if df <= 30 {
		tValue = tValues[25]
	} else if df <= 40 {
		tValue = tValues[30]
	} else if df <= 60 {
		tValue = tValues[40]
	} else if df <= 120 {
		tValue = tValues[100]
	}

	// Calculate confidence interval
	margin := tValue * se
	confLower := mean - margin
	confUpper := mean + margin

	// Find min and max
	minVal := values[0]
	maxVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	return &SteadyStateStats{
		CPUMean:         mean,
		CPUStdDev:       stdDev,
		CPUConfLower:    confLower,
		CPUConfUpper:    confUpper,
		CPUMin:          minVal,
		CPUMax:          maxVal,
		SampleSize:      n,
		ConfidenceLevel: confidenceLevel,
	}
}

// sqrt calculates square root using Newton's method
func sqrt(x float64) float64 {
	if x == 0 {
		return 0
	}
	if x < 0 {
		return 0 // Return 0 for negative numbers (shouldn't happen in our case)
	}

	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}
