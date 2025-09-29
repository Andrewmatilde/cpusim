package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	collectorAPI "cpusim/collector/api/generated"
	dashboardAPI "cpusim/dashboard/api/generated"
	"cpusim/dashboard/pkg/config"
)

type DashboardService struct {
	config *config.Config
}

func NewDashboardService(cfg *config.Config) *DashboardService {
	return &DashboardService{
		config: cfg,
	}
}

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

	health := &dashboardAPI.HostHealth{
		Name: &hostName,
		Ip:   &host.IP,
	}

	// 检查CPU服务 - 通过HTTP请求测试连通性
	cpuURL := host.GetCPUServiceURL() + "/calculate"
	_, err := http.Post(cpuURL, "application/json", strings.NewReader(`{"a":1,"b":1}`))
	health.CpuServiceHealthy = boolPtr(err == nil)

	// 检查Collector服务
	collectorClient, err := collectorAPI.NewClientWithResponses(host.GetCollectorServiceURL())
	if err != nil {
		health.CollectorServiceHealthy = boolPtr(false)
	} else {
		resp, err := collectorClient.HealthCheckWithResponse(ctx)
		if err != nil || resp.StatusCode() != 200 {
			health.CollectorServiceHealthy = boolPtr(false)
		} else {
			health.CollectorServiceHealthy = boolPtr(true)
			if resp.JSON200 != nil {
				health.CollectorHealth = &struct {
					Status    *string `json:"status,omitempty"`
					Timestamp *string `json:"timestamp,omitempty"`
				}{
					Status:    stringPtr(string(resp.JSON200.Status)),
					Timestamp: stringPtr(resp.JSON200.Timestamp.String()),
				}
			}
		}
	}

	return health, nil
}

func (s *DashboardService) TestHostCalculation(ctx context.Context, hostName string, req dashboardAPI.CalculationRequest) (*dashboardAPI.CalculationResponse, error) {
	host := s.config.GetHostByName(hostName)
	if host == nil {
		return nil, fmt.Errorf("host not found: %s", hostName)
	}

	// 直接通过HTTP调用CPU服务的/calculate接口
	cpuURL := host.GetCPUServiceURL() + "/calculate"

	// 构造请求体
	requestBody := map[string]interface{}{}
	if req.A != nil {
		requestBody["a"] = *req.A
	}
	if req.B != nil {
		requestBody["b"] = *req.B
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

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

func (s *DashboardService) GetHostExperiments(ctx context.Context, hostName string) (*dashboardAPI.ExperimentListResponse, error) {
	host := s.config.GetHostByName(hostName)
	if host == nil {
		return nil, fmt.Errorf("host not found: %s", hostName)
	}

	client, err := collectorAPI.NewClientWithResponses(host.GetCollectorServiceURL())
	if err != nil {
		return nil, fmt.Errorf("failed to create collector client: %w", err)
	}

	resp, err := client.ListExperimentsWithResponse(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiments: %w", err)
	}

	if resp.StatusCode() != 200 || resp.JSON200 == nil {
		return nil, fmt.Errorf("get experiments failed with status: %d", resp.StatusCode())
	}

	// 转换响应
	var experiments []dashboardAPI.ExperimentStatus
	for _, exp := range resp.JSON200.Experiments {
		dashExp := dashboardAPI.ExperimentStatus{
			ExperimentId:        stringPtr(exp.ExperimentId),
			Status:              stringPtr(string(exp.Status)),
			IsActive:            &exp.IsActive,
			DataPointsCollected: exp.DataPointsCollected,
			StartTime:           stringPtr(exp.StartTime.String()),
		}
		if exp.EndTime != nil {
			dashExp.EndTime = stringPtr(exp.EndTime.String())
		}
		if exp.Duration != nil {
			dashExp.Duration = exp.Duration
		}
		// TODO: Fix LastMetrics conversion later
		experiments = append(experiments, dashExp)
	}

	return &dashboardAPI.ExperimentListResponse{
		Experiments: &experiments,
		Total:       &resp.JSON200.Total,
		HasMore:     resp.JSON200.HasMore,
	}, nil
}

func (s *DashboardService) StartHostExperiment(ctx context.Context, hostName string, req dashboardAPI.ExperimentRequest) (*dashboardAPI.ExperimentResponse, error) {
	host := s.config.GetHostByName(hostName)
	if host == nil {
		return nil, fmt.Errorf("host not found: %s", hostName)
	}

	client, err := collectorAPI.NewClientWithResponses(host.GetCollectorServiceURL())
	if err != nil {
		return nil, fmt.Errorf("failed to create collector client: %w", err)
	}

	// 如果没有提供experimentId，生成一个合法的k8s名称
	experimentId := req.ExperimentId
	if experimentId == "" {
		// 生成类似k8s名称的ID
		experimentId = fmt.Sprintf("exp-%d", time.Now().Unix())
	}

	collectorReq := collectorAPI.StartExperimentJSONRequestBody{
		ExperimentId:       experimentId,
		Description:        &req.Description,
		Timeout:            &req.Timeout,
		CollectionInterval: &req.CollectionInterval,
	}

	resp, err := client.StartExperimentWithResponse(ctx, collectorReq)
	if err != nil {
		return nil, fmt.Errorf("failed to start experiment: %w", err)
	}

	if resp.StatusCode() != 200 || resp.JSON200 == nil {
		return nil, fmt.Errorf("start experiment failed with status: %d", resp.StatusCode())
	}

	return &dashboardAPI.ExperimentResponse{
		ExperimentId: stringPtr(resp.JSON200.ExperimentId),
		Message:      resp.JSON200.Message,
		Status:       stringPtr(string(resp.JSON200.Status)),
		Timestamp:    stringPtr(resp.JSON200.Timestamp.String()),
	}, nil
}

func (s *DashboardService) GetHostExperimentData(ctx context.Context, hostName, experimentId string) (*dashboardAPI.ExperimentData, error) {
	host := s.config.GetHostByName(hostName)
	if host == nil {
		return nil, fmt.Errorf("host not found: %s", hostName)
	}

	client, err := collectorAPI.NewClientWithResponses(host.GetCollectorServiceURL())
	if err != nil {
		return nil, fmt.Errorf("failed to create collector client: %w", err)
	}

	// Parse experimentId as string (no need to parse UUID anymore)
	resp, err := client.GetExperimentDataWithResponse(ctx, experimentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment data: %w", err)
	}

	if resp.StatusCode() != 200 || resp.JSON200 == nil {
		return nil, fmt.Errorf("get experiment data failed with status: %d", resp.StatusCode())
	}

	exp := resp.JSON200
	dashExp := &dashboardAPI.ExperimentData{
		ExperimentId:       stringPtr(exp.ExperimentId),
		Description:        exp.Description,
		StartTime:          stringPtr(exp.StartTime.String()),
		CollectionInterval: exp.CollectionInterval,
	}

	if exp.EndTime != nil {
		dashExp.EndTime = stringPtr(exp.EndTime.String())
	}
	if exp.Duration != nil {
		dashExp.Duration = exp.Duration
	}

	// Convert metrics
	if exp.Metrics != nil {
		var metrics []dashboardAPI.MetricDataPoint
		for _, metric := range exp.Metrics {
			dashMetric := dashboardAPI.MetricDataPoint{
				Timestamp: stringPtr(metric.Timestamp.String()),
				SystemMetrics: &struct {
					CalculatorServiceHealthy *bool `json:"calculatorServiceHealthy,omitempty"`
					CpuUsagePercent          *float32 `json:"cpuUsagePercent,omitempty"`
					MemoryUsageBytes         *int `json:"memoryUsageBytes,omitempty"`
					MemoryUsagePercent       *float32 `json:"memoryUsagePercent,omitempty"`
					NetworkIOBytes           *struct {
						BytesReceived   *int `json:"bytesReceived,omitempty"`
						BytesSent       *int `json:"bytesSent,omitempty"`
						PacketsReceived *int `json:"packetsReceived,omitempty"`
						PacketsSent     *int `json:"packetsSent,omitempty"`
					} `json:"networkIOBytes,omitempty"`
				}{
					CalculatorServiceHealthy: &metric.SystemMetrics.CalculatorServiceHealthy,
					CpuUsagePercent:          &metric.SystemMetrics.CpuUsagePercent,
					MemoryUsageBytes:         int64ToIntPtr(metric.SystemMetrics.MemoryUsageBytes),
					MemoryUsagePercent:       &metric.SystemMetrics.MemoryUsagePercent,
					NetworkIOBytes: &struct {
						BytesReceived   *int `json:"bytesReceived,omitempty"`
						BytesSent       *int `json:"bytesSent,omitempty"`
						PacketsReceived *int `json:"packetsReceived,omitempty"`
						PacketsSent     *int `json:"packetsSent,omitempty"`
					}{
						BytesReceived:   int64ToIntPtr(metric.SystemMetrics.NetworkIOBytes.BytesReceived),
						BytesSent:       int64ToIntPtr(metric.SystemMetrics.NetworkIOBytes.BytesSent),
						PacketsReceived: int64ToIntPtr(metric.SystemMetrics.NetworkIOBytes.PacketsReceived),
						PacketsSent:     int64ToIntPtr(metric.SystemMetrics.NetworkIOBytes.PacketsSent),
					},
				},
			}
			metrics = append(metrics, dashMetric)
		}
		dashExp.Metrics = &metrics
	}

	return dashExp, nil
}

func (s *DashboardService) GetHostExperimentStatus(ctx context.Context, hostName, experimentId string) (*dashboardAPI.ExperimentStatus, error) {
	host := s.config.GetHostByName(hostName)
	if host == nil {
		return nil, fmt.Errorf("host not found: %s", hostName)
	}

	client, err := collectorAPI.NewClientWithResponses(host.GetCollectorServiceURL())
	if err != nil {
		return nil, fmt.Errorf("failed to create collector client: %w", err)
	}

	// Parse experimentId as string (no need to parse UUID anymore)
	resp, err := client.GetExperimentStatusWithResponse(ctx, experimentId)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment status: %w", err)
	}

	if resp.StatusCode() != 200 || resp.JSON200 == nil {
		return nil, fmt.Errorf("get experiment status failed with status: %d", resp.StatusCode())
	}

	exp := resp.JSON200
	dashExp := &dashboardAPI.ExperimentStatus{
		ExperimentId:        stringPtr(exp.ExperimentId),
		Status:              stringPtr(string(exp.Status)),
		IsActive:            &exp.IsActive,
		DataPointsCollected: exp.DataPointsCollected,
		StartTime:           stringPtr(exp.StartTime.String()),
	}

	if exp.EndTime != nil {
		dashExp.EndTime = stringPtr(exp.EndTime.String())
	}
	if exp.Duration != nil {
		dashExp.Duration = exp.Duration
	}
	if exp.LastMetrics != nil {
		dashExp.LastMetrics = &struct {
			CalculatorServiceHealthy *bool    `json:"calculatorServiceHealthy,omitempty"`
			CpuUsagePercent          *float32 `json:"cpuUsagePercent,omitempty"`
			MemoryUsageBytes         *int     `json:"memoryUsageBytes,omitempty"`
			MemoryUsagePercent       *float32 `json:"memoryUsagePercent,omitempty"`
			NetworkIOBytes           *struct {
				BytesReceived   *int `json:"bytesReceived,omitempty"`
				BytesSent       *int `json:"bytesSent,omitempty"`
				PacketsReceived *int `json:"packetsReceived,omitempty"`
				PacketsSent     *int `json:"packetsSent,omitempty"`
			} `json:"networkIOBytes,omitempty"`
		}{
			CalculatorServiceHealthy: &exp.LastMetrics.CalculatorServiceHealthy,
			CpuUsagePercent:          &exp.LastMetrics.CpuUsagePercent,
			MemoryUsageBytes:         int64ToIntPtr(exp.LastMetrics.MemoryUsageBytes),
			MemoryUsagePercent:       &exp.LastMetrics.MemoryUsagePercent,
			NetworkIOBytes: &struct {
				BytesReceived   *int `json:"bytesReceived,omitempty"`
				BytesSent       *int `json:"bytesSent,omitempty"`
				PacketsReceived *int `json:"packetsReceived,omitempty"`
				PacketsSent     *int `json:"packetsSent,omitempty"`
			}{
				BytesReceived:   int64ToIntPtr(exp.LastMetrics.NetworkIOBytes.BytesReceived),
				BytesSent:       int64ToIntPtr(exp.LastMetrics.NetworkIOBytes.BytesSent),
				PacketsReceived: int64ToIntPtr(exp.LastMetrics.NetworkIOBytes.PacketsReceived),
				PacketsSent:     int64ToIntPtr(exp.LastMetrics.NetworkIOBytes.PacketsSent),
			},
		}
	}

	return dashExp, nil
}

func (s *DashboardService) StopHostExperiment(ctx context.Context, hostName, experimentId string) (*dashboardAPI.ExperimentResponse, error) {
	host := s.config.GetHostByName(hostName)
	if host == nil {
		return nil, fmt.Errorf("host not found: %s", hostName)
	}

	client, err := collectorAPI.NewClientWithResponses(host.GetCollectorServiceURL())
	if err != nil {
		return nil, fmt.Errorf("failed to create collector client: %w", err)
	}

	// Parse experimentId as string (no need to parse UUID anymore)
	resp, err := client.StopExperimentWithResponse(ctx, experimentId)
	if err != nil {
		return nil, fmt.Errorf("failed to stop experiment: %w", err)
	}

	if resp.StatusCode() != 200 || resp.JSON200 == nil {
		return nil, fmt.Errorf("stop experiment failed with status: %d", resp.StatusCode())
	}

	return &dashboardAPI.ExperimentResponse{
		ExperimentId: stringPtr(resp.JSON200.ExperimentId),
		Message:      resp.JSON200.Message,
		Status:       stringPtr(string(resp.JSON200.Status)),
		Timestamp:    stringPtr(resp.JSON200.Timestamp.String()),
	}, nil
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

func int64ToIntPtr(i int64) *int {
	intVal := int(i)
	return &intVal
}