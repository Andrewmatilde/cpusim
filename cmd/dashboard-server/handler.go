package main

import (
	"net/http"
	"time"

	"cpusim/dashboard/api/generated"
	"cpusim/pkg/dashboard"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// APIHandler implements the OpenAPI generated ServerInterface
type APIHandler struct {
	service *dashboard.Service
	config  dashboard.Config
	logger  zerolog.Logger
}

// GetServiceConfig implements getting the service configuration
func (h *APIHandler) GetServiceConfig(c *gin.Context) {
	response := generated.ServiceConfig{
		TargetHosts: convertTargetHostsToAPI(h.config.TargetHosts),
		ClientHost:  convertClientHostToAPI(h.config.ClientHost),
	}
	c.JSON(http.StatusOK, response)
}

// GetStatus implements getting the current status
func (h *APIHandler) GetStatus(c *gin.Context) {
	status := h.service.GetStatus()
	response := generated.StatusResponse{
		Status:    status,
		Timestamp: time.Now(),
	}
	c.JSON(http.StatusOK, response)
}

// StartExperiment implements starting a new dashboard experiment
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

	timeout := time.Duration(request.Timeout) * time.Second

	err := h.service.StartExperiment(request.ExperimentId, timeout)
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "internal_error"

		if err.Error() == "cannot start experiment: current status is Running, must be Pending" {
			statusCode = http.StatusConflict
			errorCode = "experiment_running"
		}

		c.JSON(statusCode, generated.ErrorResponse{
			Error:     errorCode,
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	response := generated.ExperimentResponse{
		ExperimentId: request.ExperimentId,
		Status:       "started",
		Timestamp:    time.Now(),
		Message:      "Dashboard experiment started successfully",
	}

	c.JSON(http.StatusOK, response)
}

// ListExperiments implements listing all stored experiments
func (h *APIHandler) ListExperiments(c *gin.Context) {
	experiments, err := h.service.ListExperiments()
	if err != nil {
		c.JSON(http.StatusInternalServerError, generated.ErrorResponse{
			Error:     "internal_error",
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Convert to API types
	apiExperiments := make([]generated.ExperimentInfo, len(experiments))
	for i, exp := range experiments {
		apiExperiments[i] = generated.ExperimentInfo{
			Id:         exp.ID,
			CreatedAt:  exp.CreatedAt,
			ModifiedAt: exp.ModifiedAt,
			FileSizeKB: exp.FileSizeKB,
		}
	}

	response := generated.ExperimentListResponse{
		Experiments: apiExperiments,
		Total:       len(apiExperiments),
		Timestamp:   time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// StopExperiment implements stopping the running experiment
func (h *APIHandler) StopExperiment(c *gin.Context, experimentId string) {
	err := h.service.StopExperiment()
	if err != nil {
		statusCode := http.StatusInternalServerError
		errorCode := "internal_error"

		if err.Error() == "cannot stop experiment: current status is Pending, must be Running" {
			statusCode = http.StatusConflict
			errorCode = "no_experiment_running"
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
			Status:       "stopped",
			Timestamp:    time.Now(),
			Message:      "Experiment stopped successfully",
		}
		c.JSON(http.StatusOK, response)
		return
	}

	response := generated.ExperimentResponse{
		ExperimentId: experimentId,
		Status:       "stopped",
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

	// Convert ExperimentData to API response
	response := generated.ExperimentData{
		Config:           convertConfigToAPI(data.Config),
		StartTime:        data.StartTime,
		EndTime:          data.EndTime,
		Duration:         float32(data.Duration),
		Status:           data.Status,
		CollectorResults: convertCollectorResultsToAPI(data.CollectorResults),
		RequesterResult:  convertRequesterResultToAPI(data.RequesterResult),
		Errors:           convertErrorsToAPI(data.Errors),
	}

	c.JSON(http.StatusOK, response)
}

// GetHostsStatus implements querying status of all hosts
func (h *APIHandler) GetHostsStatus(c *gin.Context) {
	targetHostsStatus, clientHostStatus, err := h.service.GetHostsStatus(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, generated.ErrorResponse{
			Error:     "internal_error",
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Convert to API types
	apiTargetHosts := make([]generated.TargetHostStatus, len(targetHostsStatus))
	for i, status := range targetHostsStatus {
		apiTargetHosts[i] = generated.TargetHostStatus{
			Name:                status.Name,
			Status:              status.Status,
			CurrentExperimentId: status.CurrentExperimentID,
			Error:               status.Error,
		}
	}

	apiClientHost := generated.ClientHostStatus{
		Name:                clientHostStatus.Name,
		Status:              clientHostStatus.Status,
		CurrentExperimentId: clientHostStatus.CurrentExperimentID,
		Error:               clientHostStatus.Error,
	}

	response := generated.HostsStatusResponse{
		TargetHostsStatus: apiTargetHosts,
		ClientHostStatus:  apiClientHost,
		Timestamp:         time.Now(),
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

// Helper functions to convert between internal and API types

func convertConfigToAPI(cfg dashboard.Config) generated.ServiceConfig {
	return generated.ServiceConfig{
		TargetHosts: convertTargetHostsToAPI(cfg.TargetHosts),
		ClientHost:  convertClientHostToAPI(cfg.ClientHost),
	}
}

func convertTargetHostsToAPI(hosts []dashboard.TargetHost) []generated.TargetHost {
	apiHosts := make([]generated.TargetHost, len(hosts))
	for i, host := range hosts {
		apiHosts[i] = generated.TargetHost{
			Name:                host.Name,
			ExternalIP:          host.ExternalIP,
			InternalIP:          host.InternalIP,
			CpuServiceURL:       host.CPUServiceURL,
			CollectorServiceURL: host.CollectorServiceURL,
		}
	}
	return apiHosts
}

func convertClientHostToAPI(host dashboard.ClientHost) generated.ClientHost {
	return generated.ClientHost{
		Name:                host.Name,
		ExternalIP:          host.ExternalIP,
		InternalIP:          host.InternalIP,
		RequesterServiceURL: host.RequesterServiceURL,
	}
}

func convertCollectorResultsToAPI(results map[string]dashboard.CollectorResult) map[string]generated.CollectorResult {
	apiResults := make(map[string]generated.CollectorResult)
	for key, result := range results {
		apiResults[key] = generated.CollectorResult{
			HostName:            result.HostName,
			ExperimentId:        result.ExperimentID,
			Status:              result.Status,
			DataPointsCollected: result.DataPointsCollected,
			Error:               result.Error,
		}
	}
	return apiResults
}

func convertRequesterResultToAPI(result *dashboard.RequesterResult) generated.RequesterResult {
	if result == nil {
		return generated.RequesterResult{}
	}
	return generated.RequesterResult{
		ExperimentId:    result.ExperimentID,
		Status:          result.Status,
		TotalRequests:   result.TotalRequests,
		Successful:      result.Successful,
		Failed:          result.Failed,
		AvgResponseTime: result.AvgResponseTime,
		Error:           result.Error,
	}
}

func convertErrorsToAPI(errors []dashboard.ExperimentError) []generated.ExperimentError {
	apiErrors := make([]generated.ExperimentError, len(errors))
	for i, err := range errors {
		apiErrors[i] = generated.ExperimentError{
			Timestamp: err.Timestamp,
			Phase:     err.Phase,
			HostName:  err.HostName,
			Message:   err.Message,
		}
	}
	return apiErrors
}
