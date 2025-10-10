package experiment

import (
	"context"
	"fmt"

	dashboardAPI "cpusim/dashboard/api/generated"
	requesterAPI "cpusim/requester/api/generated"
)

// GetData retrieves experiment data
func (em *ExperimentManager) GetData(ctx context.Context, experimentID string, params dashboardAPI.GetExperimentDataParams) (*dashboardAPI.ExperimentDataResponse, error) {
	// Get the experiment
	experiment, err := em.Get(ctx, experimentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment: %w", err)
	}

	response := &dashboardAPI.ExperimentDataResponse{
		ExperimentId: experimentID,
	}

	// Collect data from all target hosts
	targetHosts := make([]struct {
		CollectorData dashboardAPI.CollectorExperimentData `json:"collectorData,omitempty"`
		ExternalIP    string                               `json:"externalIP"`
		InternalIP    string                               `json:"internalIP,omitempty"`
		Name          string                               `json:"name"`
	}, 0, len(experiment.TargetHosts))

	for _, targetHost := range experiment.TargetHosts {
		hostData := struct {
			CollectorData dashboardAPI.CollectorExperimentData `json:"collectorData,omitempty"`
			ExternalIP    string                               `json:"externalIP"`
			InternalIP    string                               `json:"internalIP,omitempty"`
			Name          string                               `json:"name"`
		}{
			Name:       targetHost.Name,
			ExternalIP: targetHost.ExternalIP,
			InternalIP: targetHost.InternalIP,
		}

		// Try to get collector data
		collectorData, err := em.targetMgr.GetCollectorExperimentData(ctx, targetHost.Name, experimentID)
		if err == nil && collectorData != nil {
			// CollectorExperimentData now directly references collector API ExperimentData
			// So we can directly assign it
			hostData.CollectorData = *collectorData
		}

		targetHosts = append(targetHosts, hostData)
	}
	response.TargetHosts = targetHosts

	// Get requester data from client host
	if experiment.ClientHost.Name != "" {
		requesterStats, err := em.clientMgr.GetRequesterStats(ctx, experiment.ClientHost.Name, experimentID)
		if err == nil && requesterStats != nil {
			response.ClientHost = struct {
				ExternalIP    string `json:"externalIP"`
				InternalIP    string `json:"internalIP,omitempty"`
				Name          string `json:"name"`
				RequesterData struct {
					Stats requesterAPI.RequestExperimentStats `json:"stats,omitempty"`
				} `json:"requesterData,omitempty"`
			}{
				Name:       experiment.ClientHost.Name,
				ExternalIP: experiment.ClientHost.ExternalIP,
				InternalIP: experiment.ClientHost.InternalIP,
			}

			// RequesterData now has a Stats field that directly references requester API stats
			response.ClientHost.RequesterData.Stats = *requesterStats
		}
	}

	return response, nil
}
