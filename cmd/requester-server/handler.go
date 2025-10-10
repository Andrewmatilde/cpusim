package main

import (
	"net/http"
	"time"

	"cpusim/requester/api/generated"
	"cpusim/requester/pkg/experiment"

	"github.com/gin-gonic/gin"
)

// APIHandler implements the OpenAPI generated ServerInterface
type APIHandler struct {
	experimentManager *experiment.Manager
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
	var statusFilter *string
	if params.Status != "" {
		status := string(params.Status)
		statusFilter = &status
	}

	experiments := h.experimentManager.ListExperiments(statusFilter)

	response := generated.RequestExperimentListResponse{
		Experiments: experiments,
		Total:       len(experiments),
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

	experiment, err := h.experimentManager.StartExperiment(request)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorType := "internal_error"

		if err.Error() == "experiment already exists" {
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

	c.JSON(http.StatusCreated, experiment)
}

// GetRequestExperimentStatus implements getting experiment status
func (h *APIHandler) GetRequestExperimentStatus(c *gin.Context, experimentId string) {
	experiment, err := h.experimentManager.GetExperiment(experimentId)
	if err != nil {
		c.JSON(http.StatusNotFound, generated.ErrorResponse{
			Error:        "experiment_not_found",
			Message:      err.Error(),
			Timestamp:    time.Now(),
			ExperimentId: experimentId,
		})
		return
	}

	c.JSON(http.StatusOK, experiment)
}

// StopRequestExperiment implements stopping an experiment
func (h *APIHandler) StopRequestExperiment(c *gin.Context, experimentId string) {
	result, err := h.experimentManager.StopExperiment(experimentId)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorType := "internal_error"

		if err.Error() == "experiment not found" {
			statusCode = http.StatusNotFound
			errorType = "experiment_not_found"
		} else if err.Error() == "experiment already stopped" {
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

	c.JSON(http.StatusOK, result)
}

// GetRequestExperimentStats implements getting experiment statistics
func (h *APIHandler) GetRequestExperimentStats(c *gin.Context, experimentId string) {
	stats, err := h.experimentManager.GetExperimentStats(experimentId)
	if err != nil {
		c.JSON(http.StatusNotFound, generated.ErrorResponse{
			Error:        "experiment_not_found",
			Message:      err.Error(),
			Timestamp:    time.Now(),
			ExperimentId: experimentId,
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}