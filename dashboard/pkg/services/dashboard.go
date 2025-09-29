package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	collectorAPI "cpusim/collector/api/generated"
	dashboardAPI "cpusim/dashboard/api/generated"
	"cpusim/dashboard/pkg/config"
)

type DashboardService struct {
	config  *config.Config
	dataDir string
}

func NewDashboardService(cfg *config.Config) *DashboardService {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./data"
	}

	// Ensure data directory exists
	os.MkdirAll(dataDir, 0755)

	return &DashboardService{
		config:  cfg,
		dataDir: dataDir,
	}
}

// Global experiment management
func (s *DashboardService) GetExperiments(ctx context.Context, params dashboardAPI.GetExperimentsParams) (*dashboardAPI.ExperimentListResponse, error) {
	// Read all experiment directories from filesystem
	entries, err := os.ReadDir(s.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			// No experiments yet
			return &dashboardAPI.ExperimentListResponse{
				Experiments: []dashboardAPI.Experiment{},
				Total:       0,
			}, nil
		}
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	experiments := []dashboardAPI.Experiment{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Load experiment metadata
		metadataFile := filepath.Join(s.dataDir, entry.Name(), "experiment.json")
		data, err := os.ReadFile(metadataFile)
		if err != nil {
			// Skip if no metadata file
			continue
		}

		var experiment dashboardAPI.Experiment
		if err := json.Unmarshal(data, &experiment); err != nil {
			// Skip invalid experiment data
			continue
		}

		experiments = append(experiments, experiment)
	}

	// Sort experiments by creation time (newest first)
	sort.Slice(experiments, func(i, j int) bool {
		return experiments[i].CreatedAt.After(experiments[j].CreatedAt)
	})

	// Apply limit if specified
	limit := 20
	if params.Limit != nil {
		limit = *params.Limit
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
		HasMore:     &hasMore,
	}, nil
}

func (s *DashboardService) CreateGlobalExperiment(ctx context.Context, req dashboardAPI.CreateExperimentRequest) (*dashboardAPI.Experiment, error) {
	// Check if experiment already exists
	expDir := filepath.Join(s.dataDir, req.ExperimentId)
	if _, err := os.Stat(expDir); err == nil {
		return nil, fmt.Errorf("experiment %s already exists", req.ExperimentId)
	}

	// Create experiment
	now := time.Now()
	experiment := &dashboardAPI.Experiment{
		ExperimentId:       req.ExperimentId,
		Description:       req.Description,
		CreatedAt:         now,
		Timeout:           req.Timeout,
		CollectionInterval: req.CollectionInterval,
		ParticipatingHosts: req.ParticipatingHosts,
	}

	// Start experiment on all participating hosts
	var wg sync.WaitGroup
	errors := make(chan error, len(req.ParticipatingHosts))

	for _, host := range req.ParticipatingHosts {
		wg.Add(1)
		go func(h struct {
			Ip   string `json:"ip"`
			Name string `json:"name"`
		}) {
			defer wg.Done()

			// Get host configuration
			hostConfig := s.config.GetHostByName(h.Name)
			if hostConfig == nil {
				errors <- fmt.Errorf("host %s not found in configuration", h.Name)
				return
			}

			// Create collector client
			client, err := collectorAPI.NewClientWithResponses(hostConfig.GetCollectorServiceURL())
			if err != nil {
				errors <- fmt.Errorf("failed to create collector client for %s: %w", h.Name, err)
				return
			}

			// Start experiment on this host
			collectorReq := collectorAPI.StartExperimentJSONRequestBody{
				ExperimentId:       req.ExperimentId,
				Description:        req.Description,
				Timeout:           req.Timeout,
				CollectionInterval: req.CollectionInterval,
			}

			resp, err := client.StartExperimentWithResponse(ctx, collectorReq)
			if err != nil {
				errors <- fmt.Errorf("failed to start experiment on %s: %w", h.Name, err)
				return
			}

			if resp.StatusCode() != 200 {
				errors <- fmt.Errorf("start experiment on %s failed with status %d", h.Name, resp.StatusCode())
			}
		}(host)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	var errList []error
	for err := range errors {
		if err != nil {
			errList = append(errList, err)
		}
	}

	if len(errList) > 0 {
		// Rollback: stop experiment on hosts where it was started
		s.rollbackExperiment(ctx, req.ExperimentId, req.ParticipatingHosts)
		return nil, fmt.Errorf("failed to start experiment on some hosts: %v", errList)
	}

	// Create experiment directory and save metadata
	os.MkdirAll(expDir, 0755)

	metadataFile := filepath.Join(expDir, "experiment.json")
	metadataBytes, _ := json.MarshalIndent(experiment, "", "  ")
	if err := os.WriteFile(metadataFile, metadataBytes, 0644); err != nil {
		// Rollback if failed to save metadata
		s.rollbackExperiment(ctx, req.ExperimentId, req.ParticipatingHosts)
		os.RemoveAll(expDir)
		return nil, fmt.Errorf("failed to save experiment metadata: %w", err)
	}

	return experiment, nil
}

func (s *DashboardService) GetGlobalExperiment(ctx context.Context, experimentId string) (*dashboardAPI.Experiment, error) {
	// Always load from disk
	metadataFile := filepath.Join(s.dataDir, experimentId, "experiment.json")
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("experiment %s not found", experimentId)
		}
		return nil, fmt.Errorf("failed to read experiment metadata: %w", err)
	}

	var experiment dashboardAPI.Experiment
	if err := json.Unmarshal(data, &experiment); err != nil {
		return nil, fmt.Errorf("invalid experiment metadata: %w", err)
	}

	return &experiment, nil
}

func (s *DashboardService) GetExperimentData(ctx context.Context, experimentId string, params dashboardAPI.GetExperimentDataParams) (*dashboardAPI.ExperimentDataResponse, error) {
	// Get experiment metadata
	experiment, err := s.GetGlobalExperiment(ctx, experimentId)
	if err != nil {
		return nil, err
	}

	response := &dashboardAPI.ExperimentDataResponse{
		ExperimentId: experimentId,
		Experiment:   experiment,
	}

	// If specific host requested, get data from that host
	if params.HostName != nil && *params.HostName != "" {
		hostName := *params.HostName

		// Find host in experiment
		var hostFound bool
		for _, h := range experiment.ParticipatingHosts {
			if h.Name == hostName {
				hostFound = true
				break
			}
		}

		if !hostFound {
			return nil, fmt.Errorf("host %s not participating in experiment %s", hostName, experimentId)
		}

		// Try to load data from disk first
		dataFile := filepath.Join(s.dataDir, experimentId, fmt.Sprintf("%s.json", hostName))
		if data, err := os.ReadFile(dataFile); err == nil {
			var collectorData collectorAPI.ExperimentData
			if err := json.Unmarshal(data, &collectorData); err == nil {
				hosts := []struct {
					Data *dashboardAPI.CollectorExperimentData `json:"data,omitempty"`
					Ip   string                              `json:"ip"`
					Name string                              `json:"name"`
				}{
					{
						Name: hostName,
						Ip:   "", // Get from config
						Data: convertCollectorDataToDashboard(&collectorData),
					},
				}

				// Get IP from config
				if hostConfig := s.config.GetHostByName(hostName); hostConfig != nil {
					hosts[0].Ip = hostConfig.IP
				}

				response.Hosts = &hosts
				return response, nil
			}
		}

		// If not on disk, try to get from collector service
		hostConfig := s.config.GetHostByName(hostName)
		if hostConfig != nil {
			client, _ := collectorAPI.NewClientWithResponses(hostConfig.GetCollectorServiceURL())
			if resp, err := client.GetExperimentDataWithResponse(ctx, experimentId); err == nil && resp.StatusCode() == 200 {
				hosts := []struct {
					Data *dashboardAPI.CollectorExperimentData `json:"data,omitempty"`
					Ip   string                              `json:"ip"`
					Name string                              `json:"name"`
				}{
					{
						Name: hostName,
						Ip:   hostConfig.IP,
						Data: convertCollectorDataToDashboard(resp.JSON200),
					},
				}
				response.Hosts = &hosts
			}
		}
	} else {
		// Return summary for all hosts
		var hosts []struct {
			Data *dashboardAPI.CollectorExperimentData `json:"data,omitempty"`
			Ip   string                              `json:"ip"`
			Name string                              `json:"name"`
		}

		for _, h := range experiment.ParticipatingHosts {
			hostInfo := struct {
				Data *dashboardAPI.CollectorExperimentData `json:"data,omitempty"`
				Ip   string                              `json:"ip"`
				Name string                              `json:"name"`
			}{
				Name: h.Name,
				Ip:   h.Ip,
			}

			// Check if data exists on disk
			dataFile := filepath.Join(s.dataDir, experimentId, fmt.Sprintf("%s.json", h.Name))
			if _, err := os.Stat(dataFile); err == nil {
				// Data exists but don't load it for summary view
				hostInfo.Data = nil
			}

			hosts = append(hosts, hostInfo)
		}

		response.Hosts = &hosts
	}

	return response, nil
}

func (s *DashboardService) StopGlobalExperiment(ctx context.Context, experimentId string) (*dashboardAPI.StopAndCollectResponse, error) {
	// Get experiment
	experiment, err := s.GetGlobalExperiment(ctx, experimentId)
	if err != nil {
		return nil, err
	}

	response := &dashboardAPI.StopAndCollectResponse{
		ExperimentId: experimentId,
		Status:       "success",
		Timestamp:    time.Now(),
	}

	var hostsCollected []struct {
		Ip   *string `json:"ip,omitempty"`
		Name *string `json:"name,omitempty"`
	}
	var hostsFailed []struct {
		Error *string `json:"error,omitempty"`
		Ip    *string `json:"ip,omitempty"`
		Name  *string `json:"name,omitempty"`
	}

	// Stop experiment on all hosts and collect data
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, host := range experiment.ParticipatingHosts {
		wg.Add(1)
		go func(h struct {
			Ip   string `json:"ip"`
			Name string `json:"name"`
		}) {
			defer wg.Done()

			hostConfig := s.config.GetHostByName(h.Name)
			if hostConfig == nil {
				mu.Lock()
				hostsFailed = append(hostsFailed, struct {
					Error *string `json:"error,omitempty"`
					Ip    *string `json:"ip,omitempty"`
					Name  *string `json:"name,omitempty"`
				}{
					Name:  stringPtr(h.Name),
					Ip:    stringPtr(h.Ip),
					Error: stringPtr("host not found in configuration"),
				})
				mu.Unlock()
				return
			}

			client, err := collectorAPI.NewClientWithResponses(hostConfig.GetCollectorServiceURL())
			if err != nil {
				mu.Lock()
				hostsFailed = append(hostsFailed, struct {
					Error *string `json:"error,omitempty"`
					Ip    *string `json:"ip,omitempty"`
					Name  *string `json:"name,omitempty"`
				}{
					Name:  stringPtr(h.Name),
					Ip:    stringPtr(h.Ip),
					Error: stringPtr(fmt.Sprintf("failed to create client: %v", err)),
				})
				mu.Unlock()
				return
			}

			// Stop experiment
			stopResp, err := client.StopExperimentWithResponse(ctx, experimentId)
			if err != nil || stopResp.StatusCode() != 200 {
				mu.Lock()
				hostsFailed = append(hostsFailed, struct {
					Error *string `json:"error,omitempty"`
					Ip    *string `json:"ip,omitempty"`
					Name  *string `json:"name,omitempty"`
				}{
					Name:  stringPtr(h.Name),
					Ip:    stringPtr(h.Ip),
					Error: stringPtr(fmt.Sprintf("failed to stop experiment: %v", err)),
				})
				mu.Unlock()
				return
			}

			// Get experiment data
			dataResp, err := client.GetExperimentDataWithResponse(ctx, experimentId)
			if err != nil || dataResp.StatusCode() != 200 {
				mu.Lock()
				hostsFailed = append(hostsFailed, struct {
					Error *string `json:"error,omitempty"`
					Ip    *string `json:"ip,omitempty"`
					Name  *string `json:"name,omitempty"`
				}{
					Name:  stringPtr(h.Name),
					Ip:    stringPtr(h.Ip),
					Error: stringPtr(fmt.Sprintf("failed to get data: %v", err)),
				})
				mu.Unlock()
				return
			}

			// Save data to disk
			expDir := filepath.Join(s.dataDir, experimentId)
			os.MkdirAll(expDir, 0755)

			dataFile := filepath.Join(expDir, fmt.Sprintf("%s.json", h.Name))
			dataBytes, _ := json.MarshalIndent(dataResp.JSON200, "", "  ")
			if err := os.WriteFile(dataFile, dataBytes, 0644); err != nil {
				mu.Lock()
				hostsFailed = append(hostsFailed, struct {
					Error *string `json:"error,omitempty"`
					Ip    *string `json:"ip,omitempty"`
					Name  *string `json:"name,omitempty"`
				}{
					Name:  stringPtr(h.Name),
					Ip:    stringPtr(h.Ip),
					Error: stringPtr(fmt.Sprintf("failed to save data: %v", err)),
				})
				mu.Unlock()
				return
			}

			mu.Lock()
			hostsCollected = append(hostsCollected, struct {
				Ip   *string `json:"ip,omitempty"`
				Name *string `json:"name,omitempty"`
			}{
				Name: stringPtr(h.Name),
				Ip:   stringPtr(h.Ip),
			})
			mu.Unlock()
		}(host)
	}

	wg.Wait()

	response.HostsCollected = &hostsCollected
	response.HostsFailed = &hostsFailed

	if len(hostsFailed) > 0 {
		if len(hostsCollected) > 0 {
			response.Status = "partial"
			response.Message = stringPtr(fmt.Sprintf("Collected data from %d hosts, failed on %d hosts", len(hostsCollected), len(hostsFailed)))
		} else {
			response.Status = "failed"
			response.Message = stringPtr("Failed to collect data from all hosts")
		}
	} else {
		response.Message = stringPtr(fmt.Sprintf("Successfully stopped experiment and collected data from %d hosts", len(hostsCollected)))
	}

	// Create consolidated data file
	s.createConsolidatedDataFile(experimentId)

	// Update experiment metadata to mark as completed
	s.markExperimentCompleted(experimentId)

	return response, nil
}

// Host management (existing methods)
func (s *DashboardService) GetHosts() []dashboardAPI.Host {
	var hosts []dashboardAPI.Host
	for _, h := range s.config.GetAllHosts() {
		hosts = append(hosts, dashboardAPI.Host{
			Name:                &h.Name,
			Ip:                  &h.IP,
			CpuServiceUrl:       stringPtr(h.GetCPUServiceURL()),
			CollectorServiceUrl: stringPtr(h.GetCollectorServiceURL()),
		})
	}
	return hosts
}

func (s *DashboardService) GetHostHealth(ctx context.Context, hostName string) (*dashboardAPI.HostHealth, error) {
	host := s.config.GetHostByName(hostName)
	if host == nil {
		return nil, fmt.Errorf("host not found: %s", hostName)
	}

	now := time.Now()
	health := &dashboardAPI.HostHealth{
		Name:      &hostName,
		Ip:        &host.IP,
		Timestamp: &now,
	}

	// Check CPU service
	cpuURL := host.GetCPUServiceURL() + "/calculate"
	req, _ := http.NewRequest("POST", cpuURL, nil)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 2 * time.Second}
	_, err := client.Do(req)
	health.CpuServiceHealthy = boolPtr(err == nil)

	// Check Collector service
	collectorClient, err := collectorAPI.NewClientWithResponses(host.GetCollectorServiceURL())
	if err != nil {
		health.CollectorServiceHealthy = boolPtr(false)
	} else {
		resp, err := collectorClient.HealthCheckWithResponse(ctx)
		health.CollectorServiceHealthy = boolPtr(err == nil && resp.StatusCode() == 200)
	}

	return health, nil
}

func (s *DashboardService) TestHostCalculation(ctx context.Context, hostName string, req dashboardAPI.CalculationRequest) (*dashboardAPI.CalculationResponse, error) {
	host := s.config.GetHostByName(hostName)
	if host == nil {
		return nil, fmt.Errorf("host not found: %s", hostName)
	}

	// Call CPU service
	cpuURL := host.GetCPUServiceURL() + "/calculate"

	requestBody := map[string]interface{}{}
	if req.A != nil {
		requestBody["a"] = *req.A
	}
	if req.B != nil {
		requestBody["b"] = *req.B
	}

	body, _ := json.Marshal(requestBody)
	resp, err := http.Post(cpuURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to call calculation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("calculation failed with status: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	gcd, _ := result["gcd"].(float64)
	processTime, _ := result["processTime"].(float64)

	gcdStr := fmt.Sprintf("%.0f", gcd)
	processTimeStr := fmt.Sprintf("%.6f", processTime)

	return &dashboardAPI.CalculationResponse{
		Gcd:         &gcdStr,
		ProcessTime: &processTimeStr,
	}, nil
}

// Helper functions
func (s *DashboardService) rollbackExperiment(ctx context.Context, experimentId string, hosts []struct {
	Ip   string `json:"ip"`
	Name string `json:"name"`
}) {
	for _, host := range hosts {
		hostConfig := s.config.GetHostByName(host.Name)
		if hostConfig != nil {
			client, _ := collectorAPI.NewClientWithResponses(hostConfig.GetCollectorServiceURL())
			client.StopExperimentWithResponse(ctx, experimentId)
		}
	}
}

func (s *DashboardService) createConsolidatedDataFile(experimentId string) error {
	expDir := filepath.Join(s.dataDir, experimentId)

	// Read experiment metadata
	metadataFile := filepath.Join(expDir, "experiment.json")
	metadataBytes, err := os.ReadFile(metadataFile)
	if err != nil {
		return err
	}

	var experiment dashboardAPI.Experiment
	if err := json.Unmarshal(metadataBytes, &experiment); err != nil {
		return err
	}

	// Collect all host data
	var hosts []map[string]interface{}
	for _, h := range experiment.ParticipatingHosts {
		dataFile := filepath.Join(expDir, fmt.Sprintf("%s.json", h.Name))
		if dataBytes, err := os.ReadFile(dataFile); err == nil {
			var data map[string]interface{}
			if err := json.Unmarshal(dataBytes, &data); err == nil {
				hostData := map[string]interface{}{
					"name": h.Name,
					"ip":   h.Ip,
					"data": data,
				}
				hosts = append(hosts, hostData)
			}
		}
	}

	// Create consolidated data
	consolidatedData := map[string]interface{}{
		"experiment": experiment,
		"hosts":      hosts,
	}

	// Save consolidated data
	dataFile := filepath.Join(expDir, "data.json")
	dataBytes, _ := json.MarshalIndent(consolidatedData, "", "  ")
	return os.WriteFile(dataFile, dataBytes, 0644)
}

func (s *DashboardService) markExperimentCompleted(experimentId string) error {
	metadataFile := filepath.Join(s.dataDir, experimentId, "experiment.json")
	data, err := os.ReadFile(metadataFile)
	if err != nil {
		return err
	}

	var experiment map[string]interface{}
	if err := json.Unmarshal(data, &experiment); err != nil {
		return err
	}

	// Add completion timestamp
	experiment["completedAt"] = time.Now().Format(time.RFC3339)

	// Save updated metadata
	metadataBytes, _ := json.MarshalIndent(experiment, "", "  ")
	return os.WriteFile(metadataFile, metadataBytes, 0644)
}

func convertCollectorDataToDashboard(data *collectorAPI.ExperimentData) *dashboardAPI.CollectorExperimentData {
	if data == nil {
		return nil
	}

	result := &dashboardAPI.CollectorExperimentData{
		ExperimentId: &data.ExperimentId,
		Description:  data.Description,
	}

	// Handle time fields
	startTime := data.StartTime
	result.StartTime = &startTime

	if data.EndTime != nil {
		result.EndTime = data.EndTime
	}
	if data.Duration != nil {
		result.Duration = data.Duration
	}
	if data.CollectionInterval != nil {
		result.CollectionInterval = data.CollectionInterval
	}

	// Convert metrics
	if data.Metrics != nil {
		var metrics []dashboardAPI.MetricDataPoint
		for _, m := range data.Metrics {
			metric := dashboardAPI.MetricDataPoint{
				Timestamp: m.Timestamp,
				SystemMetrics: struct {
					CalculatorServiceHealthy bool    `json:"calculatorServiceHealthy"`
					CpuUsagePercent          float32 `json:"cpuUsagePercent"`
					MemoryUsageBytes         int64   `json:"memoryUsageBytes"`
					MemoryUsagePercent       float32 `json:"memoryUsagePercent"`
					NetworkIOBytes           struct {
						BytesReceived   int64 `json:"bytesReceived"`
						BytesSent       int64 `json:"bytesSent"`
						PacketsReceived int64 `json:"packetsReceived"`
						PacketsSent     int64 `json:"packetsSent"`
					} `json:"networkIOBytes"`
				}{
					CpuUsagePercent:          m.SystemMetrics.CpuUsagePercent,
					MemoryUsageBytes:         m.SystemMetrics.MemoryUsageBytes,
					MemoryUsagePercent:       m.SystemMetrics.MemoryUsagePercent,
					CalculatorServiceHealthy: m.SystemMetrics.CalculatorServiceHealthy,
					NetworkIOBytes: struct {
						BytesReceived   int64 `json:"bytesReceived"`
						BytesSent       int64 `json:"bytesSent"`
						PacketsReceived int64 `json:"packetsReceived"`
						PacketsSent     int64 `json:"packetsSent"`
					}{
						BytesReceived:   m.SystemMetrics.NetworkIOBytes.BytesReceived,
						BytesSent:       m.SystemMetrics.NetworkIOBytes.BytesSent,
						PacketsReceived: m.SystemMetrics.NetworkIOBytes.PacketsReceived,
						PacketsSent:     m.SystemMetrics.NetworkIOBytes.PacketsSent,
					},
				},
			}
			metrics = append(metrics, metric)
		}
		result.Metrics = &metrics
	}

	return result
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}