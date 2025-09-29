package main

import (
	"net/http"
	"time"

	"cpusim/collector/api/generated"
	"cpusim/collector/pkg/experiment"
	"cpusim/collector/pkg/storage"

	"github.com/gin-gonic/gin"
)

// APIHandler implements the generated OpenAPI interface
type APIHandler struct {
	experimentManager *experiment.Manager
	storage          *storage.FileStorage
}

// ListExperiments returns a list of all experiments
func (h *APIHandler) ListExperiments(c *gin.Context, params generated.ListExperimentsParams) {
	// Set defaults
	limit := 50
	if params.Limit != nil {
		limit = *params.Limit
	}

	status := generated.ListExperimentsParamsStatusAll
	if params.Status != nil {
		status = *params.Status
	}

	// Get all experiments from manager (active ones)
	experiments := h.getAllExperiments()

	// Filter by status if specified
	var filteredExperiments []generated.ExperimentSummary
	for _, exp := range experiments {
		if shouldIncludeExperiment(exp, status) {
			filteredExperiments = append(filteredExperiments, exp)
		}
	}

	// Apply limit
	total := len(filteredExperiments)
	hasMore := false
	if len(filteredExperiments) > limit {
		filteredExperiments = filteredExperiments[:limit]
		hasMore = true
	}

	response := generated.ExperimentListResponse{
		Experiments: filteredExperiments,
		Total:       total,
		HasMore:     &hasMore,
	}

	c.JSON(http.StatusOK, response)
}

// getAllExperiments gets all experiments from both manager and storage
func (h *APIHandler) getAllExperiments() []generated.ExperimentSummary {
	var experiments []generated.ExperimentSummary

	// Get active experiments from manager
	activeSummaries := h.experimentManager.ListAllExperiments()
	for _, summary := range activeSummaries {
		expUUID := summary.ID

		var statusEnum generated.ExperimentSummaryStatus
		switch summary.Status {
		case experiment.StatusRunning:
			statusEnum = generated.ExperimentSummaryStatusRunning
		case experiment.StatusStopped:
			statusEnum = generated.ExperimentSummaryStatusStopped
		case experiment.StatusTimeout:
			statusEnum = generated.ExperimentSummaryStatusTimeout
		default:
			statusEnum = generated.ExperimentSummaryStatusError
		}

		apiSummary := generated.ExperimentSummary{
			ExperimentId: expUUID,
			Status:       statusEnum,
			StartTime:    summary.StartTime,
			IsActive:     summary.IsActive,
		}

		if summary.Description != "" {
			apiSummary.Description = &summary.Description
		}

		if summary.EndTime != nil {
			apiSummary.EndTime = summary.EndTime
			apiSummary.Duration = summary.Duration
		}

		if summary.DataPointsCollected > 0 {
			apiSummary.DataPointsCollected = &summary.DataPointsCollected
		}

		experiments = append(experiments, apiSummary)
	}

	// Get experiments from storage (completed ones not in manager)
	if storedExperiments, err := h.storage.ListExperiments(); err == nil {
		for _, stored := range storedExperiments {
			// Check if this experiment is already in active list
			found := false
			for _, active := range activeSummaries {
				if active.ID == stored.ExperimentID {
					found = true
					break
				}
			}
			if found {
				continue // Skip if already included from active experiments
			}

			expUUID := stored.ExperimentID

			// Try to load the experiment data to get more details
			if data, err := h.storage.LoadExperimentData(stored.ExperimentID); err == nil {
				var statusEnum generated.ExperimentSummaryStatus
				if data.EndTime != nil {
					statusEnum = generated.ExperimentSummaryStatusStopped
				} else {
					statusEnum = generated.ExperimentSummaryStatusError
				}

				summary := generated.ExperimentSummary{
					ExperimentId: expUUID,
					Status:       statusEnum,
					StartTime:    data.StartTime,
					IsActive:     false,
				}

				if data.Description != "" {
					summary.Description = &data.Description
				}

				if data.EndTime != nil {
					summary.EndTime = data.EndTime
					summary.Duration = &data.Duration
				}

				dataPointsCount := len(data.Metrics)
				summary.DataPointsCollected = &dataPointsCount

				experiments = append(experiments, summary)
			}
		}
	}

	return experiments
}

// shouldIncludeExperiment checks if experiment should be included based on status filter
func shouldIncludeExperiment(exp generated.ExperimentSummary, statusFilter generated.ListExperimentsParamsStatus) bool {
	if statusFilter == generated.ListExperimentsParamsStatusAll {
		return true
	}

	switch statusFilter {
	case generated.ListExperimentsParamsStatusRunning:
		return exp.Status == generated.ExperimentSummaryStatusRunning
	case generated.ListExperimentsParamsStatusStopped:
		return exp.Status == generated.ExperimentSummaryStatusStopped
	case generated.ListExperimentsParamsStatusTimeout:
		return exp.Status == generated.ExperimentSummaryStatusTimeout
	case generated.ListExperimentsParamsStatusError:
		return exp.Status == generated.ExperimentSummaryStatusError
	default:
		return true
	}
}

// StartExperiment starts a new data collection experiment
func (h *APIHandler) StartExperiment(c *gin.Context) {
	var req generated.StartExperimentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, generated.ErrorResponse{
			Error:     "invalid_request",
			Message:   "Invalid request body: " + err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Convert string to string (no conversion needed)
	experimentID := req.ExperimentId

	// Set defaults
	collectionInterval := time.Duration(defaultCollectionInterval) * time.Millisecond
	if req.CollectionInterval != nil {
		collectionInterval = time.Duration(*req.CollectionInterval) * time.Millisecond
	}

	timeout := time.Duration(defaultTimeout) * time.Second
	if req.Timeout != nil {
		timeout = time.Duration(*req.Timeout) * time.Second
	}

	description := ""
	if req.Description != nil {
		description = *req.Description
	}

	// Start experiment
	exp, err := h.experimentManager.StartExperiment(
		experimentID,
		description,
		collectionInterval,
		timeout,
	)
	if err != nil {
		status := http.StatusInternalServerError
		errorCode := "start_failed"

		// Check for specific error types
		if err.Error() == "experiment with ID "+experimentID+" already exists" {
			status = http.StatusConflict
			errorCode = "experiment_exists"
		}

		c.JSON(status, generated.ErrorResponse{
			Error:     errorCode,
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	expUUID := exp.ID
	message := "Experiment started successfully"
	c.JSON(http.StatusOK, generated.ExperimentResponse{
		ExperimentId: expUUID,
		Status:       generated.ExperimentResponseStatusStarted,
		Timestamp:    exp.StartTime,
		Message:      &message,
	})
}

// StopExperiment stops an active experiment
func (h *APIHandler) StopExperiment(c *gin.Context, experimentId string) {
	experimentID := experimentId

	exp, err := h.experimentManager.StopExperiment(experimentID)
	if err != nil {
		status := http.StatusInternalServerError
		errorCode := "stop_failed"

		if err.Error() == "experiment with ID "+experimentID+" not found" {
			status = http.StatusNotFound
			errorCode = "experiment_not_found"
		}

		c.JSON(status, generated.ErrorResponse{
			Error:     errorCode,
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	message := "Experiment stopped successfully"
	status := generated.ExperimentResponseStatusStopped
	if exp.Status == experiment.StatusTimeout {
		message = "Experiment stopped due to timeout"
		status = generated.ExperimentResponseStatusTimeout
	}

	expUUID := exp.ID
	c.JSON(http.StatusOK, generated.ExperimentResponse{
		ExperimentId: expUUID,
		Status:       status,
		Timestamp:    time.Now(),
		Message:      &message,
	})
}

// GetExperimentStatus returns the current status of an experiment
func (h *APIHandler) GetExperimentStatus(c *gin.Context, experimentId string) {
	experimentID := experimentId

	exp, err := h.experimentManager.GetExperiment(experimentID)
	if err != nil {
		c.JSON(http.StatusNotFound, generated.ErrorResponse{
			Error:     "experiment_not_found",
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	expUUID := exp.ID
	dataPointsCollected := exp.DataPointsCollected

	var statusEnum generated.ExperimentStatusStatus
	switch exp.Status {
	case experiment.StatusRunning:
		statusEnum = generated.ExperimentStatusStatusRunning
	case experiment.StatusStopped:
		statusEnum = generated.ExperimentStatusStatusStopped
	case experiment.StatusTimeout:
		statusEnum = generated.ExperimentStatusStatusTimeout
	default:
		statusEnum = generated.ExperimentStatusStatusError
	}

	status := generated.ExperimentStatus{
		ExperimentId:        expUUID,
		Status:              statusEnum,
		StartTime:           exp.StartTime,
		IsActive:            exp.IsActive,
		DataPointsCollected: &dataPointsCollected,
	}

	if exp.EndTime != nil {
		status.EndTime = exp.EndTime
		duration := int(exp.EndTime.Sub(exp.StartTime).Seconds())
		status.Duration = &duration
	}

	if exp.LastMetrics != nil {
		status.LastMetrics = &generated.SystemMetrics{
			CpuUsagePercent:          float32(exp.LastMetrics.CPUUsagePercent),
			MemoryUsageBytes:         exp.LastMetrics.MemoryUsageBytes,
			MemoryUsagePercent:       float32(exp.LastMetrics.MemoryUsagePercent),
			CalculatorServiceHealthy: exp.LastMetrics.CalculatorServiceHealthy,
			NetworkIOBytes: generated.NetworkIO{
				BytesReceived:   exp.LastMetrics.NetworkIOBytes.BytesReceived,
				BytesSent:       exp.LastMetrics.NetworkIOBytes.BytesSent,
				PacketsReceived: exp.LastMetrics.NetworkIOBytes.PacketsReceived,
				PacketsSent:     exp.LastMetrics.NetworkIOBytes.PacketsSent,
			},
		}
	}

	c.JSON(http.StatusOK, status)
}

// GetExperimentData returns the collected data for an experiment
func (h *APIHandler) GetExperimentData(c *gin.Context, experimentId string) {
	experimentID := experimentId

	data, err := h.experimentManager.GetExperimentData(experimentID)
	if err != nil {
		c.JSON(http.StatusNotFound, generated.ErrorResponse{
			Error:     "experiment_not_found",
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Convert to API format
	expUUID := data.ExperimentID
	collectionInterval := data.CollectionInterval
	result := generated.ExperimentData{
		ExperimentId:       expUUID,
		StartTime:          data.StartTime,
		CollectionInterval: &collectionInterval,
		Metrics:            make([]generated.MetricDataPoint, 0, len(data.Metrics)),
	}

	if data.Description != "" {
		result.Description = &data.Description
	}

	if data.EndTime != nil {
		result.EndTime = data.EndTime
		result.Duration = &data.Duration
	}

	// Convert metrics
	for _, metric := range data.Metrics {
		dataPoint := generated.MetricDataPoint{
			Timestamp: metric.Timestamp,
			SystemMetrics: generated.SystemMetrics{
				CpuUsagePercent:          float32(metric.SystemMetrics.CPUUsagePercent),
				MemoryUsageBytes:         metric.SystemMetrics.MemoryUsageBytes,
				MemoryUsagePercent:       float32(metric.SystemMetrics.MemoryUsagePercent),
				CalculatorServiceHealthy: metric.SystemMetrics.CalculatorServiceHealthy,
				NetworkIOBytes: generated.NetworkIO{
					BytesReceived:   metric.SystemMetrics.NetworkIOBytes.BytesReceived,
					BytesSent:       metric.SystemMetrics.NetworkIOBytes.BytesSent,
					PacketsReceived: metric.SystemMetrics.NetworkIOBytes.PacketsReceived,
					PacketsSent:     metric.SystemMetrics.NetworkIOBytes.PacketsSent,
				},
			},
		}
		result.Metrics = append(result.Metrics, dataPoint)
	}

	c.JSON(http.StatusOK, result)
}

// HealthCheck returns the health status of the collector service
func (h *APIHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, generated.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    nil, // Could be implemented to track actual uptime
	})
}