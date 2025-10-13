package dashboard

import (
	"context"
	collectorAPI "cpusim/collector/api/generated"
	requesterAPI "cpusim/requester/api/generated"
	"fmt"
	"time"

	"cpusim/pkg/exp"
	"github.com/rs/zerolog"
)

// Service manages dashboard experiments using the exp framework
type Service struct {
	exp.Manager[*ExperimentData]

	fs     exp.FileStorage[*ExperimentData]
	logger zerolog.Logger
	config Config

	// HTTP clients for sub-experiments
	collectorClients map[string]CollectorClient // key: host name
	requesterClient  RequesterClient

	// Current experiment ID and QPS
	currentExperimentID string
	currentQPS          int
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

	s := &Service{
		fs:               *fs,
		logger:           logger,
		config:           config,
		collectorClients: make(map[string]CollectorClient),
	}

	// Create collector function
	collectFunc := func(ctx context.Context) (*ExperimentData, error) {
		return s.runExperiment(ctx)
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

	// Store the experiment ID and QPS so runExperiment can use them
	s.currentExperimentID = id
	s.currentQPS = qps

	return s.Manager.Start(id, timeout)
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
func (s *Service) runExperiment(ctx context.Context) (*ExperimentData, error) {
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
			s.StopAll(s.currentExperimentID)
			return data, err
		}

		// Start collector experiment
		// Use the experiment timeout from the context or a default value
		timeout := 60 * time.Second
		if deadline, ok := ctx.Deadline(); ok {
			timeout = time.Until(deadline)
		}
		if err := client.StartExperiment(ctx, s.currentExperimentID, timeout); err != nil {
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
			s.StopAll(s.currentExperimentID)
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
		s.StopAll(s.currentExperimentID)
		return data, err
	}

	// Use the experiment timeout from the context or a default value
	timeout := 60 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		timeout = time.Until(deadline)
	}
	if err := s.requesterClient.StartExperiment(ctx, s.currentExperimentID, timeout, s.currentQPS); err != nil {
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
		s.StopAll(s.currentExperimentID)
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
		if err := client.StopExperiment(stopCtx, s.currentExperimentID); err != nil {
			s.logger.Warn().Err(err).Str("host", hostName).Msg("Failed to stop collector")
		} else {
			s.logger.Info().Str("host", hostName).Msg("Collector stopped successfully")
		}
	}

	// Stop requester
	if err := s.requesterClient.StopExperiment(stopCtx, s.currentExperimentID); err != nil {
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
		if collectorData, err := client.GetExperiment(collectCtx, s.currentExperimentID); err == nil {
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
	if requesterStats, err := s.requesterClient.GetExperiment(collectCtx, s.currentExperimentID); err == nil {
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
