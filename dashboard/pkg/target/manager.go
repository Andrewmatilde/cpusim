package target

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	collectorAPI "cpusim/collector/api/generated"
	"cpusim/dashboard/pkg/config"
	dashboardAPI "cpusim/dashboard/api/generated"
)

// TargetManager implements the Manager interface for managing target hosts
type TargetManager struct {
	config     *config.Config
	httpClient *http.Client
}

// NewManager creates a new TargetManager
func NewManager(cfg *config.Config) *TargetManager {
	return &TargetManager{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// ListTargets returns all configured target hosts
func (tm *TargetManager) ListTargets() []*Target {
	var targets []*Target
	for _, host := range tm.config.GetAllHosts() {
		if host.IsTarget() {
			targets = append(targets, &Target{
				Name:       host.Name,
				ExternalIP: host.ExternalIP,
				InternalIP: host.InternalIP,
			})
		}
	}
	return targets
}

// GetTarget returns a target by name
func (tm *TargetManager) GetTarget(name string) (*Target, error) {
	host := tm.config.GetHostByName(name)
	if host == nil {
		return nil, fmt.Errorf("target not found: %s", name)
	}
	if !host.IsTarget() {
		return nil, fmt.Errorf("host %s is not a target", name)
	}
	return &Target{
		Name:       host.Name,
		ExternalIP: host.ExternalIP,
		InternalIP: host.InternalIP,
	}, nil
}

// CheckHealth checks the health of a specific target
func (tm *TargetManager) CheckHealth(ctx context.Context, targetName string) (*TargetHealth, error) {
	target, err := tm.GetTarget(targetName)
	if err != nil {
		return nil, err
	}

	health := &TargetHealth{
		Name:        target.Name,
		ExternalIP:  target.ExternalIP,
		InternalIP:  target.InternalIP,
		LastChecked: time.Now(),
	}

	// Check CPU service
	cpuURL := target.GetCPUServiceURL() + "/calculate"
	req, _ := http.NewRequestWithContext(ctx, "POST", cpuURL, nil)
	req.Header.Set("Content-Type", "application/json")

	_, err = tm.httpClient.Do(req)
	health.CPUServiceHealthy = (err == nil)

	// Check Collector service
	collectorClient, err := collectorAPI.NewClientWithResponses(target.GetCollectorServiceURL())
	if err != nil {
		health.CollectorServiceHealthy = false
	} else {
		resp, err := collectorClient.HealthCheckWithResponse(ctx)
		health.CollectorServiceHealthy = (err == nil && resp.StatusCode() == 200)
	}

	return health, nil
}

// CheckAllHealth checks the health of all targets
func (tm *TargetManager) CheckAllHealth(ctx context.Context) (map[string]*TargetHealth, error) {
	targets := tm.ListTargets()
	healthMap := make(map[string]*TargetHealth)

	for _, target := range targets {
		health, err := tm.CheckHealth(ctx, target.Name)
		if err != nil {
			// Log error but continue checking other targets
			fmt.Printf("Failed to check health for target %s: %v\n", target.Name, err)
			continue
		}
		healthMap[target.Name] = health
	}

	return healthMap, nil
}

// StartCollector starts the collector service on a target
func (tm *TargetManager) StartCollector(ctx context.Context, targetName, experimentID, description string) error {
	target, err := tm.GetTarget(targetName)
	if err != nil {
		return err
	}

	client, err := collectorAPI.NewClientWithResponses(target.GetCollectorServiceURL())
	if err != nil {
		return fmt.Errorf("failed to create collector client: %w", err)
	}

	reqBody := collectorAPI.StartExperimentRequest{
		ExperimentId: experimentID,
		Description:  description,
	}

	resp, err := client.StartExperimentWithResponse(ctx, reqBody)
	if err != nil {
		return fmt.Errorf("failed to start collector: %w", err)
	}

	if resp.StatusCode() != 200 && resp.StatusCode() != 409 {
		return fmt.Errorf("collector returned error: HTTP %d", resp.StatusCode())
	}

	// 409 means experiment already running, which is acceptable
	if resp.StatusCode() == 409 {
		fmt.Printf("Collector on %s: experiment already running (HTTP 409)\n", targetName)
	}

	return nil
}

// StopCollector stops the collector service on a target and retrieves data
func (tm *TargetManager) StopCollector(ctx context.Context, targetName, experimentID string) (*CollectorData, error) {
	target, err := tm.GetTarget(targetName)
	if err != nil {
		return nil, err
	}

	client, err := collectorAPI.NewClientWithResponses(target.GetCollectorServiceURL())
	if err != nil {
		return nil, fmt.Errorf("failed to create collector client: %w", err)
	}

	resp, err := client.StopExperimentWithResponse(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to stop collector: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("collector returned error: HTTP %d", resp.StatusCode())
	}

	// Retrieve data
	return tm.GetCollectorData(ctx, targetName, experimentID)
}

// GetCollectorData retrieves collected data from a target
func (tm *TargetManager) GetCollectorData(ctx context.Context, targetName, experimentID string) (*CollectorData, error) {
	target, err := tm.GetTarget(targetName)
	if err != nil {
		return nil, err
	}

	client, err := collectorAPI.NewClientWithResponses(target.GetCollectorServiceURL())
	if err != nil {
		return nil, fmt.Errorf("failed to create collector client: %w", err)
	}

	resp, err := client.GetExperimentDataWithResponse(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collector data: %w", err)
	}

	if resp.StatusCode() != 200 || resp.JSON200 == nil {
		return nil, fmt.Errorf("collector returned error: HTTP %d", resp.StatusCode())
	}

	data := resp.JSON200
	return &CollectorData{
		Duration: data.Duration,
		Metrics:  data.Metrics,
	}, nil
}

// GetCollectorExperimentData retrieves experiment data in API format from a target
func (tm *TargetManager) GetCollectorExperimentData(ctx context.Context, targetName, experimentID string) (*collectorAPI.ExperimentData, error) {
	target, err := tm.GetTarget(targetName)
	if err != nil {
		return nil, err
	}

	apiClient, err := collectorAPI.NewClientWithResponses(target.GetCollectorServiceURL())
	if err != nil {
		return nil, fmt.Errorf("failed to create collector client: %w", err)
	}

	resp, err := apiClient.GetExperimentDataWithResponse(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment data: %w", err)
	}

	if resp.StatusCode() != 200 || resp.JSON200 == nil {
		return nil, fmt.Errorf("collector returned error: HTTP %d", resp.StatusCode())
	}

	return resp.JSON200, nil
}

// TestCalculation tests the CPU calculation service on a target
func (tm *TargetManager) TestCalculation(ctx context.Context, targetName string, req dashboardAPI.CalculationRequest) (*dashboardAPI.CalculationResponse, error) {
	target, err := tm.GetTarget(targetName)
	if err != nil {
		return nil, err
	}

	cpuURL := target.GetCPUServiceURL() + "/calculate"

	requestBody := map[string]interface{}{
		"a": req.A,
		"b": req.B,
	}

	body, _ := json.Marshal(requestBody)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", cpuURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := tm.httpClient.Do(httpReq)
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

	return &dashboardAPI.CalculationResponse{
		Gcd:         fmt.Sprintf("%.0f", gcd),
		ProcessTime: fmt.Sprintf("%.6f", processTime),
	}, nil
}