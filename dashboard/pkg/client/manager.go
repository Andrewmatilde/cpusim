package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"cpusim/dashboard/pkg/config"
	requesterAPI "cpusim/requester/api/generated"
)

// ClientManager implements the Manager interface for managing client hosts
type ClientManager struct {
	config     *config.Config
	httpClient *http.Client
}

// NewManager creates a new ClientManager
func NewManager(cfg *config.Config) *ClientManager {
	return &ClientManager{
		config: cfg,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// ListClients returns all configured client hosts
func (cm *ClientManager) ListClients() []*Client {
	var clients []*Client
	for _, host := range cm.config.GetAllHosts() {
		if host.IsClient() {
			clients = append(clients, &Client{
				Name:       host.Name,
				ExternalIP: host.ExternalIP,
				InternalIP: host.InternalIP,
			})
		}
	}
	return clients
}

// GetClient returns a client by name
func (cm *ClientManager) GetClient(name string) (*Client, error) {
	host := cm.config.GetHostByName(name)
	if host == nil {
		return nil, fmt.Errorf("client not found: %s", name)
	}
	if !host.IsClient() {
		return nil, fmt.Errorf("host %s is not a client", name)
	}
	return &Client{
		Name:       host.Name,
		ExternalIP: host.ExternalIP,
		InternalIP: host.InternalIP,
	}, nil
}

// CheckHealth checks the health of a specific client
func (cm *ClientManager) CheckHealth(ctx context.Context, clientName string) (*ClientHealth, error) {
	client, err := cm.GetClient(clientName)
	if err != nil {
		return nil, err
	}

	health := &ClientHealth{
		Name:        client.Name,
		ExternalIP:  client.ExternalIP,
		InternalIP:  client.InternalIP,
		LastChecked: time.Now(),
	}

	// Check Requester service
	requesterURL := client.GetRequesterServiceURL() + "/health"
	req, _ := http.NewRequestWithContext(ctx, "GET", requesterURL, nil)

	resp, err := cm.httpClient.Do(req)
	health.RequesterServiceHealthy = (err == nil && resp != nil && resp.StatusCode == 200)
	if resp != nil {
		resp.Body.Close()
	}

	return health, nil
}

// CheckAllHealth checks the health of all clients
func (cm *ClientManager) CheckAllHealth(ctx context.Context) (map[string]*ClientHealth, error) {
	clients := cm.ListClients()
	healthMap := make(map[string]*ClientHealth)

	for _, client := range clients {
		health, err := cm.CheckHealth(ctx, client.Name)
		if err != nil {
			// Log error but continue checking other clients
			fmt.Printf("Failed to check health for client %s: %v\n", client.Name, err)
			continue
		}
		healthMap[client.Name] = health
	}

	return healthMap, nil
}

// StartRequester starts the requester service on a client
func (cm *ClientManager) StartRequester(ctx context.Context, clientName string, config RequesterConfig) error {
	client, err := cm.GetClient(clientName)
	if err != nil {
		return err
	}

	apiClient, err := requesterAPI.NewClientWithResponses(client.GetRequesterServiceURL())
	if err != nil {
		return fmt.Errorf("failed to create requester client: %w", err)
	}

	// Parse target URL to get IP and port
	var targetIP string
	var targetPort int = 80

	// Simple URL parsing (assumes http://ip:port format)
	url := config.TargetURL
	if len(url) > 7 && url[:7] == "http://" {
		url = url[7:]
	}
	// Split by colon if port is specified
	colonIdx := -1
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == ':' {
			colonIdx = i
			break
		}
	}
	if colonIdx > 0 {
		targetIP = url[:colonIdx]
		// Parse port
		fmt.Sscanf(url[colonIdx+1:], "%d", &targetPort)
	} else {
		targetIP = url
	}

	reqBody := requesterAPI.StartRequestExperimentRequest{
		ExperimentId: config.ExperimentID,
		TargetIP:     targetIP,
		TargetPort:   targetPort,
		Qps:          config.QPS,
		Timeout:      config.Duration,
	}

	resp, err := apiClient.StartRequestExperimentWithResponse(ctx, reqBody)
	if err != nil {
		return fmt.Errorf("failed to start requester: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("requester returned error: HTTP %d", resp.StatusCode())
	}

	return nil
}

// StopRequester stops the requester service on a client and retrieves data
func (cm *ClientManager) StopRequester(ctx context.Context, clientName, experimentID string) (*RequesterData, error) {
	client, err := cm.GetClient(clientName)
	if err != nil {
		return nil, err
	}

	apiClient, err := requesterAPI.NewClientWithResponses(client.GetRequesterServiceURL())
	if err != nil {
		return nil, fmt.Errorf("failed to create requester client: %w", err)
	}

	resp, err := apiClient.StopRequestExperimentWithResponse(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to stop requester: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("requester returned error: HTTP %d", resp.StatusCode())
	}

	// Retrieve data
	return cm.GetRequesterData(ctx, clientName, experimentID)
}

// GetRequesterData retrieves requester data from a client
func (cm *ClientManager) GetRequesterData(ctx context.Context, clientName, experimentID string) (*RequesterData, error) {
	client, err := cm.GetClient(clientName)
	if err != nil {
		return nil, err
	}

	apiClient, err := requesterAPI.NewClientWithResponses(client.GetRequesterServiceURL())
	if err != nil {
		return nil, fmt.Errorf("failed to create requester client: %w", err)
	}

	resp, err := apiClient.GetRequestExperimentStatsWithResponse(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get requester data: %w", err)
	}

	if resp.StatusCode() != 200 || resp.JSON200 == nil {
		return nil, fmt.Errorf("requester returned error: HTTP %d", resp.StatusCode())
	}

	stats := resp.JSON200
	endTime := stats.EndTime
	data := &RequesterData{
		TotalRequests:       stats.TotalRequests,
		SuccessfulRequests:  stats.SuccessfulRequests,
		FailedRequests:      stats.FailedRequests,
		AverageResponseTime: stats.AverageResponseTime,
		StartTime:           stats.StartTime,
		EndTime:             &endTime,
	}

	// Populate percentiles if available
	if stats.ResponseTimeP50 != 0 || stats.ResponseTimeP95 != 0 || stats.ResponseTimeP99 != 0 {
		data.Percentiles = &struct {
			P50 *float32
			P95 *float32
			P99 *float32
		}{
			P50: &stats.ResponseTimeP50,
			P95: &stats.ResponseTimeP95,
			P99: &stats.ResponseTimeP99,
		}
	}

	return data, nil
}

// GetRequesterStats retrieves requester stats in API format from a client
func (cm *ClientManager) GetRequesterStats(ctx context.Context, clientName, experimentID string) (*requesterAPI.RequestExperimentStats, error) {
	client, err := cm.GetClient(clientName)
	if err != nil {
		return nil, err
	}

	apiClient, err := requesterAPI.NewClientWithResponses(client.GetRequesterServiceURL())
	if err != nil {
		return nil, fmt.Errorf("failed to create requester client: %w", err)
	}

	resp, err := apiClient.GetRequestExperimentStatsWithResponse(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get requester stats: %w", err)
	}

	if resp.StatusCode() != 200 || resp.JSON200 == nil {
		return nil, fmt.Errorf("requester returned error: HTTP %d", resp.StatusCode())
	}

	return resp.JSON200, nil
}