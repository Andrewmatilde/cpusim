package main

import (
	"net/http"
	"time"

	"cpusim/collector/api/generated"
	"cpusim/pkg/collector"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// APIHandler implements the OpenAPI generated ServerInterface
type APIHandler struct {
	service *collector.Service
	config  collector.Config
	logger  zerolog.Logger
}

// GetServiceConfig implements getting the service configuration
func (h *APIHandler) GetServiceConfig(c *gin.Context) {
	response := generated.ServiceConfig{
		CollectionInterval: h.config.CollectionInterval,
		CalculatorProcess:  h.config.CalculatorProcess,
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
	response := generated.HealthResponse{
		Status:    "healthy",
		Timestamp: time.Now(),
		Uptime:    0, // Could be calculated from service start time
	}
	c.JSON(http.StatusOK, response)
}

// ListExperiments implements getting list of experiments
func (h *APIHandler) ListExperiments(c *gin.Context, params generated.ListExperimentsParams) {
	// Note: Current Service design only supports one experiment at a time
	// This is a simplified implementation that returns empty list
	// In the future, we can add support for storing experiment history

	response := generated.ExperimentListResponse{
		Experiments: []generated.ExperimentSummary{},
		Total:       0,
		HasMore:     false,
	}

	c.JSON(http.StatusOK, response)
}

// StartExperiment implements starting a new experiment
func (h *APIHandler) StartExperiment(c *gin.Context) {
	var request generated.StartExperimentRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, generated.ErrorResponse{
			Error:     "invalid_request",
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Convert timeout from seconds to Duration
	timeout := time.Duration(request.Timeout) * time.Second

	// Start experiment using the service
	err := h.service.StartExperiment(request.ExperimentId, timeout)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "internal_error"

		if err.Error() == "experiment already started" {
			statusCode = http.StatusConflict
			errorCode = "experiment_exists"
		}

		c.JSON(statusCode, generated.ErrorResponse{
			Error:     errorCode,
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Return experiment info
	response := generated.ExperimentResponse{
		ExperimentId: request.ExperimentId,
		Status:       generated.ExperimentResponseStatusStarted,
		Timestamp:    time.Now(),
		Message:      "Experiment started successfully",
	}

	c.JSON(http.StatusOK, response)
}

// StopExperiment implements stopping an experiment
func (h *APIHandler) StopExperiment(c *gin.Context, experimentId string) {
	// Note: Current Service design doesn't need experimentId for Stop
	// We just call Stop on the service
	err := h.service.StopExperiment()
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "internal_error"

		if err.Error() == "experiment already stopped" {
			statusCode = http.StatusConflict
			errorCode = "experiment_already_stopped"
		}

		c.JSON(statusCode, generated.ErrorResponse{
			Error:     errorCode,
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Try to get the experiment data for response
	data, err := h.service.GetExperiment(experimentId)
	if err != nil {
		// If we can't get the data, just return basic response
		response := generated.ExperimentResponse{
			ExperimentId: experimentId,
			Status:       generated.ExperimentResponseStatusStopped,
			Timestamp:    time.Now(),
			Message:      "Experiment stopped successfully",
		}
		c.JSON(http.StatusOK, response)
		return
	}

	response := generated.ExperimentResponse{
		ExperimentId: experimentId,
		Status:       generated.ExperimentResponseStatusStopped,
		Timestamp:    data.EndTime,
		Message:      "Experiment stopped successfully",
	}

	c.JSON(http.StatusOK, response)
}

// GetExperimentData implements getting experiment data
func (h *APIHandler) GetExperimentData(c *gin.Context, experimentId string) {
	data, err := h.service.GetExperiment(experimentId)
	if err != nil {
		c.JSON(http.StatusNotFound, generated.ErrorResponse{
			Error:     "experiment_not_found",
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Convert MetricsData to ExperimentData
	result := generated.ExperimentData{
		ExperimentId:       experimentId,
		StartTime:          data.StartTime,
		CollectionInterval: h.config.CollectionInterval * 1000, // Convert to milliseconds
		Metrics:            make([]generated.MetricDataPoint, 0, len(data.Metrics)),
	}

	if !data.EndTime.IsZero() {
		result.EndTime = data.EndTime
		result.Duration = int(data.Duration)
	}

	// Convert metrics
	for _, metric := range data.Metrics {
		dataPoint := generated.MetricDataPoint{
			Timestamp: metric.Timestamp,
			SystemMetrics: generated.SystemMetrics{
				CpuUsagePercent:          float32(metric.CPUUsagePercent),
				MemoryUsageBytes:         metric.MemoryUsageBytes,
				MemoryUsagePercent:       float32(metric.MemoryUsagePercent),
				CalculatorServiceHealthy: metric.CalculatorServiceHealthy,
				NetworkIOBytes: generated.NetworkIO{
					BytesReceived:   metric.NetworkIOBytes.BytesReceived,
					BytesSent:       metric.NetworkIOBytes.BytesSent,
					PacketsReceived: metric.NetworkIOBytes.PacketsReceived,
					PacketsSent:     metric.NetworkIOBytes.PacketsSent,
				},
			},
		}
		result.Metrics = append(result.Metrics, dataPoint)
	}

	c.JSON(http.StatusOK, result)
}
