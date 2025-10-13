package main

import (
	"net/http"
	"time"

	"cpusim/pkg/requester"
	"cpusim/requester/api/generated"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// APIHandler implements the OpenAPI generated ServerInterface
type APIHandler struct {
	service *requester.Service
	config  requester.Config
	logger  zerolog.Logger
}

// GetServiceConfig implements getting the service configuration
func (h *APIHandler) GetServiceConfig(c *gin.Context) {
	response := generated.ServiceConfig{
		TargetIP:   h.config.TargetIP,
		TargetPort: h.config.TargetPort,
		Qps:        h.config.QPS,
		Timeout:    h.config.Timeout,
	}
	c.JSON(http.StatusOK, response)
}

// GetStatus implements getting the service status
func (h *APIHandler) GetStatus(c *gin.Context) {
	status := h.service.GetStatus()
	currentExpID := h.service.GetCurrentExperimentID()

	response := generated.StatusResponse{
		Status:              generated.StatusResponseStatus(status),
		CurrentExperimentId: currentExpID,
	}
	c.JSON(http.StatusOK, response)
}

// HealthCheck implements the health check endpoint
func (h *APIHandler) HealthCheck(c *gin.Context) {
	now := time.Now()
	uptime := 0 // This should be calculated from service start time

	response := generated.HealthResponse{
		Status:    "healthy",
		Timestamp: now,
		Uptime:    uptime,
		Version:   "1.0.0",
	}
	c.JSON(http.StatusOK, response)
}

// ListRequestExperiments implements getting list of experiments
func (h *APIHandler) ListRequestExperiments(c *gin.Context, params generated.ListRequestExperimentsParams) {
	// Note: Current Service design only supports one experiment at a time
	// This is a simplified implementation that returns empty list
	// In the future, we can add support for storing experiment history

	response := generated.RequestExperimentListResponse{
		Experiments: []generated.RequestExperiment{},
		Total:       0,
	}

	c.JSON(http.StatusOK, response)
}

// StartRequestExperiment implements starting a new experiment
func (h *APIHandler) StartRequestExperiment(c *gin.Context) {
	var request generated.StartRequestExperimentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, generated.ErrorResponse{
			Error:        "invalid_request",
			Message:      err.Error(),
			Timestamp:    time.Now(),
			ExperimentId: request.ExperimentId,
		})
		return
	}

	// Convert timeout from seconds to Duration
	timeout := time.Duration(request.Timeout) * time.Second

	// Start experiment using the service
	err := h.service.StartExperiment(request.ExperimentId, timeout)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorType := "internal_error"

		if err.Error() == "experiment already started" {
			statusCode = http.StatusConflict
			errorType = "experiment_exists"
		}

		c.JSON(statusCode, generated.ErrorResponse{
			Error:        errorType,
			Message:      err.Error(),
			Timestamp:    time.Now(),
			ExperimentId: request.ExperimentId,
		})
		return
	}

	// Return experiment info
	experiment := generated.RequestExperiment{
		ExperimentId: request.ExperimentId,
		Description:  request.Description,
		Status:       generated.RequestExperimentStatusRunning,
		StartTime:    time.Now(),
		CreatedAt:    time.Now(),
	}

	c.JSON(http.StatusCreated, experiment)
}

// StopRequestExperiment implements stopping an experiment
func (h *APIHandler) StopRequestExperiment(c *gin.Context, experimentId string) {
	// Note: Current Service design doesn't need experimentId for Stop
	// We just call Stop on the service
	err := h.service.StopExperiment()
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorType := "internal_error"

		if err.Error() == "experiment already stopped" {
			statusCode = http.StatusConflict
			errorType = "experiment_already_stopped"
		}

		c.JSON(statusCode, generated.ErrorResponse{
			Error:        errorType,
			Message:      err.Error(),
			Timestamp:    time.Now(),
			ExperimentId: experimentId,
		})
		return
	}

	// Try to get the experiment data
	data, err := h.service.GetExperiment(experimentId)
	if err != nil {
		// If we can't get the data, just return basic stop result
		result := generated.StopExperimentResult{
			ExperimentId: experimentId,
			StopStatus:   "stopped",
			EndTime:      time.Now(),
		}
		c.JSON(http.StatusOK, result)
		return
	}

	// Return stop result with stats
	result := generated.StopExperimentResult{
		ExperimentId: experimentId,
		StopStatus:   "stopped",
		EndTime:      data.EndTime,
		Duration:     int(data.Duration),
		FinalStats: generated.RequestExperimentStats{
			ExperimentId:        experimentId,
			TotalRequests:       int(data.TotalRequests),
			SuccessfulRequests:  int(data.Successful),
			FailedRequests:      int(data.Failed),
			AverageResponseTime: float32(data.Stats.AvgResponseTime),
			MinResponseTime:     float32(data.Stats.MinResponseTime),
			MaxResponseTime:     float32(data.Stats.MaxResponseTime),
			ResponseTimeP50:     float32(data.Stats.P50),
			ResponseTimeP95:     float32(data.Stats.P95),
			ResponseTimeP99:     float32(data.Stats.P99),
			RequestsPerSecond:   float32(data.Stats.ActualQPS),
			ErrorRate:           float32(data.Stats.ErrorRate),
			StartTime:           data.StartTime,
			EndTime:             data.EndTime,
			Duration:            int(data.Duration),
			LastUpdated:         data.EndTime,
		},
	}

	c.JSON(http.StatusOK, result)
}

// GetRequestExperimentStats implements getting experiment statistics
func (h *APIHandler) GetRequestExperimentStats(c *gin.Context, experimentId string) {
	data, err := h.service.GetExperiment(experimentId)
	if err != nil {
		c.JSON(http.StatusNotFound, generated.ErrorResponse{
			Error:        "experiment_not_found",
			Message:      err.Error(),
			Timestamp:    time.Now(),
			ExperimentId: experimentId,
		})
		return
	}

	// Convert RequestData to RequestExperimentStats
	status := generated.RequestExperimentStatsStatusCompleted
	if data.EndTime.IsZero() {
		status = generated.RequestExperimentStatsStatusRunning
	}
	if data.Failed > 0 && data.Successful == 0 {
		status = generated.RequestExperimentStatsStatusError
	}

	stats := generated.RequestExperimentStats{
		ExperimentId:        experimentId,
		Status:              status,
		TotalRequests:       int(data.TotalRequests),
		SuccessfulRequests:  int(data.Successful),
		FailedRequests:      int(data.Failed),
		AverageResponseTime: float32(data.Stats.AvgResponseTime),
		MinResponseTime:     float32(data.Stats.MinResponseTime),
		MaxResponseTime:     float32(data.Stats.MaxResponseTime),
		ResponseTimeP50:     float32(data.Stats.P50),
		ResponseTimeP95:     float32(data.Stats.P95),
		ResponseTimeP99:     float32(data.Stats.P99),
		RequestsPerSecond:   float32(data.Stats.ActualQPS),
		ErrorRate:           float32(data.Stats.ErrorRate),
		StartTime:           data.StartTime,
		EndTime:             data.EndTime,
		Duration:            int(data.Duration),
		LastUpdated:         data.EndTime,
	}

	c.JSON(http.StatusOK, stats)
}
