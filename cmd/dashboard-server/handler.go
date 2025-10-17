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

	err := h.service.StartExperiment(request.ExperimentId, timeout, request.Qps)
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

// StartExperimentGroup implements starting a new experiment group
func (h *APIHandler) StartExperimentGroup(c *gin.Context) {
	var request generated.StartExperimentGroupRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, generated.ErrorResponse{
			Error:     "invalid_request",
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Create experiment group config with QPS range
	config := dashboard.ExperimentGroupConfig{
		QPSMin:       request.QpsMin,
		QPSMax:       request.QpsMax,
		QPSStep:      request.QpsStep,
		RepeatCount:  request.RepeatCount,
		Timeout:      request.Timeout,
		DelayBetween: request.DelayBetween,
	}

	// Start experiment group (this will run asynchronously)
	go func() {
		err := h.service.StartExperimentGroup(request.GroupId, request.Description, config)
		if err != nil {
			h.logger.Error().Err(err).Str("group_id", request.GroupId).Msg("Failed to start experiment group")
		}
	}()

	response := generated.ExperimentGroupResponse{
		GroupId:   request.GroupId,
		Status:    "started",
		Timestamp: time.Now(),
		Message:   "Experiment group started successfully",
	}

	c.JSON(http.StatusOK, response)
}

// ResumeExperimentGroup implements resuming an incomplete experiment group
func (h *APIHandler) ResumeExperimentGroup(c *gin.Context, groupId string) {
	// Check if service is busy
	status := h.service.GetStatus()
	if status != "Pending" {
		c.JSON(http.StatusConflict, generated.ErrorResponse{
			Error:     "service_busy",
			Message:   "Service is currently busy running another experiment group",
			Timestamp: time.Now(),
		})
		return
	}

	// Check if group exists
	group, err := h.service.GetExperimentGroup(groupId)
	if err != nil {
		c.JSON(http.StatusBadRequest, generated.ErrorResponse{
			Error:     "group_not_found",
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Check if group is already completed
	if group.Status == "completed" {
		c.JSON(http.StatusBadRequest, generated.ErrorResponse{
			Error:     "group_already_completed",
			Message:   "Cannot resume a completed experiment group",
			Timestamp: time.Now(),
		})
		return
	}

	// Resume experiment group (this will run asynchronously)
	go func() {
		err := h.service.ResumeExperimentGroup(groupId)
		if err != nil {
			h.logger.Error().Err(err).Str("group_id", groupId).Msg("Failed to resume experiment group")
		}
	}()

	response := generated.ExperimentGroupResponse{
		GroupId:   groupId,
		Status:    "resumed",
		Timestamp: time.Now(),
		Message:   "Experiment group resumed successfully",
	}

	c.JSON(http.StatusOK, response)
}

// ListExperimentGroups implements listing all experiment groups
func (h *APIHandler) ListExperimentGroups(c *gin.Context) {
	groups, err := h.service.ListExperimentGroups()
	if err != nil {
		c.JSON(http.StatusInternalServerError, generated.ErrorResponse{
			Error:     "internal_error",
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Convert to API types
	apiGroups := make([]generated.ExperimentGroup, len(groups))
	for i, group := range groups {
		apiGroups[i] = convertExperimentGroupToAPI(*group)
	}

	response := generated.ExperimentGroupListResponse{
		Groups:    apiGroups,
		Total:     len(apiGroups),
		Timestamp: time.Now(),
	}

	c.JSON(http.StatusOK, response)
}

// GetExperimentGroupWithDetails implements getting experiment group with all experiment data
func (h *APIHandler) GetExperimentGroupWithDetails(c *gin.Context, groupId string) {
	group, experiments, err := h.service.GetExperimentGroupWithDetails(groupId)
	if err != nil {
		c.JSON(http.StatusNotFound, generated.ErrorResponse{
			Error:     "group_not_found",
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return
	}

	// Convert to API types
	apiGroup := convertExperimentGroupToAPI(*group)
	apiExperiments := make([]generated.ExperimentData, len(experiments))
	for i, exp := range experiments {
		apiExperiments[i] = generated.ExperimentData{
			Config:           convertConfigToAPI(exp.Config),
			StartTime:        exp.StartTime,
			EndTime:          exp.EndTime,
			Duration:         float32(exp.Duration),
			Status:           exp.Status,
			CollectorResults: convertCollectorResultsToAPI(exp.CollectorResults),
			RequesterResult:  convertRequesterResultToAPI(exp.RequesterResult),
			Errors:           convertErrorsToAPI(exp.Errors),
		}
	}

	response := generated.ExperimentGroupDetail{
		Group:             apiGroup,
		ExperimentDetails: apiExperiments,
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
		apiResult := generated.CollectorResult{
			HostName: result.HostName,
			Status:   result.Status,
			Error:    result.Error,
		}
		// Include experiment data but clear metrics to reduce data transfer
		if result.Data != nil {
			dataCopy := *result.Data
			dataCopy.Metrics = nil // Clear time-series metrics
			apiResult.Data = dataCopy
		}
		apiResults[key] = apiResult
	}
	return apiResults
}

func convertRequesterResultToAPI(result *dashboard.RequesterResult) generated.RequesterResult {
	if result == nil {
		return generated.RequesterResult{}
	}
	apiResult := generated.RequesterResult{
		Status: result.Status,
		Error:  result.Error,
	}
	// Include complete stats if available
	if result.Stats != nil {
		apiResult.Stats = *result.Stats
	}
	return apiResult
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

func convertExperimentGroupToAPI(group dashboard.ExperimentGroup) generated.ExperimentGroup {
	// Convert QPS points with statistics
	apiQPSPoints := make([]generated.QPSPoint, len(group.QPSPoints))
	for i, qpsPoint := range group.QPSPoints {
		// Convert statistics for this QPS point
		var apiStatistics map[string]generated.SteadyStateStats
		if qpsPoint.Statistics != nil {
			apiStatistics = make(map[string]generated.SteadyStateStats)
			for hostName, stats := range qpsPoint.Statistics {
				if stats != nil {
					apiStatistics[hostName] = generated.SteadyStateStats{
						CpuMean:         float32(stats.CPUMean),
						CpuStdDev:       float32(stats.CPUStdDev),
						CpuConfLower:    float32(stats.CPUConfLower),
						CpuConfUpper:    float32(stats.CPUConfUpper),
						CpuMin:          float32(stats.CPUMin),
						CpuMax:          float32(stats.CPUMax),
						SampleSize:      stats.SampleSize,
						ConfidenceLevel: float32(stats.ConfidenceLevel),
					}
				}
			}
		}

		apiQPSPoints[i] = generated.QPSPoint{
			Qps:         qpsPoint.QPS,
			Experiments: qpsPoint.Experiments,
			Statistics:  apiStatistics,
			Status:      qpsPoint.Status,
		}
	}

	return generated.ExperimentGroup{
		GroupId:     group.GroupID,
		Description: group.Description,
		Config: generated.ExperimentGroupConfig{
			QpsMin:       group.Config.QPSMin,
			QpsMax:       group.Config.QPSMax,
			QpsStep:      group.Config.QPSStep,
			RepeatCount:  group.Config.RepeatCount,
			Timeout:      group.Config.Timeout,
			DelayBetween: group.Config.DelayBetween,
		},
		EnvironmentConfig: convertConfigToAPI(group.EnvironmentConfig),
		QpsPoints:         apiQPSPoints,
		StartTime:         group.StartTime,
		EndTime:           group.EndTime,
		Status:            group.Status,
		CurrentQPS:        group.CurrentQPS,
		CurrentRun:        group.CurrentRun,
	}
}
