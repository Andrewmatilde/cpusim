package client

import (
	"context"

	requesterAPI "cpusim/requester/api/generated"
)

// Manager defines the interface for managing client hosts
type Manager interface {
	// ListClients returns all configured client hosts
	ListClients() []*Client

	// GetClient returns a client by name
	GetClient(name string) (*Client, error)

	// CheckHealth checks the health of a specific client
	CheckHealth(ctx context.Context, clientName string) (*ClientHealth, error)

	// CheckAllHealth checks the health of all clients
	CheckAllHealth(ctx context.Context) (map[string]*ClientHealth, error)

	// StartRequester starts the requester service on a client
	StartRequester(ctx context.Context, clientName string, config RequesterConfig) error

	// StopRequester stops the requester service on a client and retrieves data
	StopRequester(ctx context.Context, clientName, experimentID string) (*RequesterData, error)

	// GetRequesterData retrieves requester data from a client
	GetRequesterData(ctx context.Context, clientName, experimentID string) (*RequesterData, error)

	// GetRequesterStats retrieves requester stats in API format
	GetRequesterStats(ctx context.Context, clientName, experimentID string) (*requesterAPI.RequestExperimentStats, error)
}