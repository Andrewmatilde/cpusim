package dashboard

import (
	"context"
	"fmt"
	"time"

	collectorAPI "cpusim/collector/api/generated"
)

// HTTPCollectorClient implements CollectorClient using HTTP API calls
type HTTPCollectorClient struct {
	client *collectorAPI.ClientWithResponses
}

// NewHTTPCollectorClient creates a new HTTP collector client
func NewHTTPCollectorClient(serverURL string) (*HTTPCollectorClient, error) {
	client, err := collectorAPI.NewClientWithResponses(serverURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create collector client: %w", err)
	}
	return &HTTPCollectorClient{
		client: client,
	}, nil
}

// StartExperiment starts a collector experiment
func (c *HTTPCollectorClient) StartExperiment(ctx context.Context, experimentID string, timeout time.Duration) error {
	timeoutSeconds := int(timeout.Seconds())

	req := collectorAPI.StartExperimentJSONRequestBody{
		ExperimentId: experimentID,
		Timeout:      timeoutSeconds,
	}

	resp, err := c.client.StartExperimentWithResponse(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to start collector experiment: %w", err)
	}

	if resp.StatusCode() != 200 {
		if resp.JSON400 != nil {
			return fmt.Errorf("collector start failed: %s", resp.JSON400.Message)
		}
		if resp.JSON409 != nil {
			return fmt.Errorf("collector start failed: %s", resp.JSON409.Message)
		}
		return fmt.Errorf("collector start failed with status %d", resp.StatusCode())
	}

	return nil
}

// StopExperiment stops a collector experiment
func (c *HTTPCollectorClient) StopExperiment(ctx context.Context, experimentID string) error {
	resp, err := c.client.StopExperimentWithResponse(ctx, experimentID)
	if err != nil {
		return fmt.Errorf("failed to stop collector experiment: %w", err)
	}

	if resp.StatusCode() != 200 {
		if resp.JSON404 != nil {
			return fmt.Errorf("collector stop failed: %s", resp.JSON404.Message)
		}
		return fmt.Errorf("collector stop failed with status %d", resp.StatusCode())
	}

	return nil
}

// GetExperiment retrieves collector experiment data
func (c *HTTPCollectorClient) GetExperiment(ctx context.Context, experimentID string) (*CollectorExperimentData, error) {
	resp, err := c.client.GetExperimentDataWithResponse(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get collector experiment data: %w", err)
	}

	if resp.StatusCode() != 200 {
		if resp.JSON404 != nil {
			return nil, fmt.Errorf("collector experiment not found: %s", resp.JSON404.Message)
		}
		return nil, fmt.Errorf("get collector experiment failed with status %d", resp.StatusCode())
	}

	if resp.JSON200 == nil {
		return nil, fmt.Errorf("no data returned from collector")
	}

	// Map the API response to our internal type
	data := &CollectorExperimentData{
		DataPointsCollected: len(resp.JSON200.Metrics),
	}

	return data, nil
}
