package dashboard

import (
	"context"
	"fmt"
	"time"

	requesterAPI "cpusim/requester/api/generated"
)

// HTTPRequesterClient implements RequesterClient using HTTP API calls
type HTTPRequesterClient struct {
	client *requesterAPI.ClientWithResponses
}

// NewHTTPRequesterClient creates a new HTTP requester client
func NewHTTPRequesterClient(serverURL string) (*HTTPRequesterClient, error) {
	client, err := requesterAPI.NewClientWithResponses(serverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create requester client: %w", err)
	}
	return &HTTPRequesterClient{
		client: client,
	}, nil
}

// StartExperiment starts a requester experiment
func (c *HTTPRequesterClient) StartExperiment(ctx context.Context, experimentID string, timeout time.Duration, qps int) error {
	timeoutSeconds := int(timeout.Seconds())

	req := requesterAPI.StartRequestExperimentJSONRequestBody{
		ExperimentId: experimentID,
		Timeout:      timeoutSeconds,
		Qps:          qps,
	}

	resp, err := c.client.StartRequestExperimentWithResponse(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start requester experiment: %w", err)
	}

	if resp.StatusCode() != 201 {
		if resp.JSON400 != nil {
			return fmt.Errorf("requester start failed: %s", resp.JSON400.Message)
		}
		if resp.JSON409 != nil {
			return fmt.Errorf("requester start failed: %s", resp.JSON409.Message)
		}
		if resp.JSON500 != nil {
			return fmt.Errorf("requester start failed: %s", resp.JSON500.Message)
		}
		return fmt.Errorf("requester start failed with status %d", resp.StatusCode())
	}

	return nil
}

// StopExperiment stops a requester experiment
func (c *HTTPRequesterClient) StopExperiment(ctx context.Context, experimentID string) error {
	resp, err := c.client.StopRequestExperimentWithResponse(ctx, experimentID)
	if err != nil {
		return fmt.Errorf("failed to stop requester experiment: %w", err)
	}

	if resp.StatusCode() != 200 {
		if resp.JSON404 != nil {
			return fmt.Errorf("requester stop failed: %s", resp.JSON404.Message)
		}
		if resp.JSON409 != nil {
			return fmt.Errorf("requester stop failed: %s", resp.JSON409.Message)
		}
		if resp.JSON500 != nil {
			return fmt.Errorf("requester stop failed: %s", resp.JSON500.Message)
		}
		return fmt.Errorf("requester stop failed with status %d", resp.StatusCode())
	}

	return nil
}

// GetExperiment retrieves requester experiment statistics
func (c *HTTPRequesterClient) GetExperiment(ctx context.Context, experimentID string) (*requesterAPI.RequestExperimentStats, error) {
	resp, err := c.client.GetRequestExperimentStatsWithResponse(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get requester experiment stats: %w", err)
	}

	if resp.StatusCode() != 200 {
		if resp.JSON404 != nil {
			return nil, fmt.Errorf("requester experiment not found: %s", resp.JSON404.Message)
		}
		if resp.JSON500 != nil {
			return nil, fmt.Errorf("get requester experiment failed: %s", resp.JSON500.Message)
		}
		return nil, fmt.Errorf("get requester experiment failed with status %d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("no data returned from requester")
	}

	return resp.JSON200, nil
}

// GetStatus retrieves the requester service status
func (c *HTTPRequesterClient) GetStatus(ctx context.Context) (string, string, error) {
	resp, err := c.client.GetStatusWithResponse(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to get requester status: %w", err)
	}

	if resp.StatusCode() != 200 {
		return "", "", fmt.Errorf("get requester status failed with status %d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return "", "", fmt.Errorf("no status returned from requester")
	}

	status := string(resp.JSON200.Status)
	experimentID := resp.JSON200.CurrentExperimentId

	return status, experimentID, nil
}
