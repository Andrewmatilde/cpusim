package handlers

import (
	"net/http"

	"cpusim/dashboard/api/generated"
	"cpusim/dashboard/pkg/services"

	"github.com/gin-gonic/gin"
)

type DashboardHandler struct {
	service *services.DashboardService
}

func NewDashboardHandler(service *services.DashboardService) *DashboardHandler {
	return &DashboardHandler{
		service: service,
	}
}

// Global experiment management endpoints
func (h *DashboardHandler) GetExperiments(c *gin.Context, params generated.GetExperimentsParams) {
	experiments, err := h.service.GetExperiments(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, experiments)
}

func (h *DashboardHandler) CreateGlobalExperiment(c *gin.Context) {
	var req generated.CreateExperimentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	experiment, err := h.service.CreateGlobalExperiment(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, experiment)
}

func (h *DashboardHandler) GetGlobalExperiment(c *gin.Context, experimentId string) {
	experiment, err := h.service.GetGlobalExperiment(c.Request.Context(), experimentId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, experiment)
}

func (h *DashboardHandler) GetExperimentData(c *gin.Context, experimentId string, params generated.GetExperimentDataParams) {
	data, err := h.service.GetExperimentData(c.Request.Context(), experimentId, params)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *DashboardHandler) StopGlobalExperiment(c *gin.Context, experimentId string) {
	result, err := h.service.StopGlobalExperiment(c.Request.Context(), experimentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Host management endpoints
func (h *DashboardHandler) GetHosts(c *gin.Context) {
	hosts := h.service.GetHosts()
	c.JSON(http.StatusOK, gin.H{"hosts": hosts})
}

func (h *DashboardHandler) GetHostHealth(c *gin.Context, name string) {
	health, err := h.service.GetHostHealth(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, health)
}

func (h *DashboardHandler) TestHostCalculation(c *gin.Context, name string) {
	var req generated.CalculationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.TestHostCalculation(c.Request.Context(), name, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// New experiment phase management endpoints
func (h *DashboardHandler) GetExperimentPhases(c *gin.Context, experimentId string) {
	phases, err := h.service.GetExperimentPhases(c.Request.Context(), experimentId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, phases)
}

func (h *DashboardHandler) StartCompleteExperiment(c *gin.Context, experimentId string) {
	result, err := h.service.StartCompleteExperiment(c.Request.Context(), experimentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *DashboardHandler) StopCompleteExperiment(c *gin.Context, experimentId string) {
	result, err := h.service.StopCompleteExperiment(c.Request.Context(), experimentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}