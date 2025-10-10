package services

import (
	"context"
	"fmt"
	"os"

	dashboardAPI "cpusim/dashboard/api/generated"
	"cpusim/dashboard/pkg/client"
	"cpusim/dashboard/pkg/config"
	"cpusim/dashboard/pkg/experiment"
	"cpusim/dashboard/pkg/target"
)

// DashboardService is a facade that delegates operations to specialized managers
type DashboardService struct {
	experimentMgr *experiment.ExperimentManager
	targetMgr     *target.TargetManager
	clientMgr     *client.ClientManager
}

// NewDashboardService creates a new DashboardService
func NewDashboardService(cfg *config.Config) *DashboardService {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data/experiments"
	}

	// Create managers
	targetMgr := target.NewManager(cfg)
	clientMgr := client.NewManager(cfg)
	experimentMgr := experiment.NewManager(targetMgr, clientMgr, dataDir)

	return &DashboardService{
		experimentMgr: experimentMgr,
		targetMgr:     targetMgr,
		clientMgr:     clientMgr,
	}
}

// ========== Experiment Operations ==========

// GetExperiments returns a list of experiments
func (s *DashboardService) GetExperiments(ctx context.Context, params dashboardAPI.GetExperimentsParams) (*dashboardAPI.ExperimentListResponse, error) {
	return s.experimentMgr.List(ctx, params)
}

// CreateGlobalExperiment creates a new global experiment
func (s *DashboardService) CreateGlobalExperiment(ctx context.Context, req dashboardAPI.CreateExperimentRequest) (*dashboardAPI.Experiment, error) {
	return s.experimentMgr.Create(ctx, req)
}

// GetGlobalExperiment retrieves a specific experiment
func (s *DashboardService) GetGlobalExperiment(ctx context.Context, experimentId string) (*dashboardAPI.Experiment, error) {
	return s.experimentMgr.Get(ctx, experimentId)
}

// GetExperimentData retrieves experiment data
func (s *DashboardService) GetExperimentData(ctx context.Context, experimentId string, params dashboardAPI.GetExperimentDataParams) (*dashboardAPI.ExperimentDataResponse, error) {
	return s.experimentMgr.GetData(ctx, experimentId, params)
}

// GetExperimentPhases retrieves experiment phase status
func (s *DashboardService) GetExperimentPhases(ctx context.Context, experimentId string) (*dashboardAPI.ExperimentPhases, error) {
	return s.experimentMgr.GetPhases(ctx, experimentId)
}

// StartCompleteExperiment starts a complete experiment (all phases)
func (s *DashboardService) StartCompleteExperiment(ctx context.Context, experimentId string) (*dashboardAPI.ExperimentOperationResponse, error) {
	return s.experimentMgr.Start(ctx, experimentId)
}

// StopCompleteExperiment stops a complete experiment (remaining phases)
func (s *DashboardService) StopCompleteExperiment(ctx context.Context, experimentId string) (*dashboardAPI.ExperimentOperationResponse, error) {
	return s.experimentMgr.Stop(ctx, experimentId)
}

// StopGlobalExperiment stops an experiment and collects data (legacy method)
func (s *DashboardService) StopGlobalExperiment(ctx context.Context, experimentId string) (*dashboardAPI.StopAndCollectResponse, error) {
	// Call the new Stop method
	result, err := s.experimentMgr.Stop(ctx, experimentId)
	if err != nil {
		return nil, err
	}

	// Convert to legacy format
	response := &dashboardAPI.StopAndCollectResponse{
		ExperimentId: experimentId,
		Status:       dashboardAPI.StopAndCollectResponseStatus(result.Status),
		Message:      result.Message,
		Timestamp:    result.Timestamp,
	}

	return response, nil
}

// ========== Host Operations ==========

// GetHosts returns all configured hosts (both targets and clients)
func (s *DashboardService) GetHosts() []dashboardAPI.Host {
	var hosts []dashboardAPI.Host

	// Add all targets
	for _, t := range s.targetMgr.ListTargets() {
		hosts = append(hosts, dashboardAPI.Host{
			Name:                t.Name,
			ExternalIP:          t.ExternalIP,
			InternalIP:          t.InternalIP,
			HostType:            config.HostTypeTarget,
			CpuServiceUrl:       t.GetCPUServiceURL(),
			CollectorServiceUrl: t.GetCollectorServiceURL(),
		})
	}

	// Add all clients
	for _, c := range s.clientMgr.ListClients() {
		hosts = append(hosts, dashboardAPI.Host{
			Name:                c.Name,
			ExternalIP:          c.ExternalIP,
			InternalIP:          c.InternalIP,
			HostType:            config.HostTypeClient,
			RequesterServiceUrl: c.GetRequesterServiceURL(),
		})
	}

	return hosts
}

// GetHostHealth retrieves health status for a specific host
func (s *DashboardService) GetHostHealth(ctx context.Context, hostName string) (*dashboardAPI.HostHealth, error) {
	// Try as target first
	if targetHealth, err := s.targetMgr.CheckHealth(ctx, hostName); err == nil {
		return &dashboardAPI.HostHealth{
			Name:                    targetHealth.Name,
			ExternalIP:              targetHealth.ExternalIP,
			InternalIP:              targetHealth.InternalIP,
			HostType:                config.HostTypeTarget,
			CpuServiceHealthy:       targetHealth.CPUServiceHealthy,
			CollectorServiceHealthy: targetHealth.CollectorServiceHealthy,
			Timestamp:               targetHealth.LastChecked,
		}, nil
	}

	// Try as client
	if clientHealth, err := s.clientMgr.CheckHealth(ctx, hostName); err == nil {
		return &dashboardAPI.HostHealth{
			Name:                    clientHealth.Name,
			ExternalIP:              clientHealth.ExternalIP,
			InternalIP:              clientHealth.InternalIP,
			HostType:                config.HostTypeClient,
			RequesterServiceHealthy: clientHealth.RequesterServiceHealthy,
			Timestamp:               clientHealth.LastChecked,
		}, nil
	}

	return nil, fmt.Errorf("host not found: %s", hostName)
}

// TestHostCalculation tests calculation on a target host
func (s *DashboardService) TestHostCalculation(ctx context.Context, hostName string, req dashboardAPI.CalculationRequest) (*dashboardAPI.CalculationResponse, error) {
	return s.targetMgr.TestCalculation(ctx, hostName, req)
}

