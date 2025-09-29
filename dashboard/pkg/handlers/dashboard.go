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

func (h *DashboardHandler) GetHosts(c *gin.Context) {
	hosts := h.service.GetHosts()
	c.JSON(http.StatusOK, hosts)
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

func (h *DashboardHandler) GetHostExperiments(c *gin.Context, name string) {
	experiments, err := h.service.GetHostExperiments(c.Request.Context(), name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, experiments)
}

func (h *DashboardHandler) StartHostExperiment(c *gin.Context, name string) {
	var req generated.ExperimentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.service.StartHostExperiment(c.Request.Context(), name, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *DashboardHandler) GetHostExperimentData(c *gin.Context, name string, experimentId string) {
	data, err := h.service.GetHostExperimentData(c.Request.Context(), name, experimentId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (h *DashboardHandler) GetHostExperimentStatus(c *gin.Context, name string, experimentId string) {
	status, err := h.service.GetHostExperimentStatus(c.Request.Context(), name, experimentId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *DashboardHandler) StopHostExperiment(c *gin.Context, name string, experimentId string) {
	result, err := h.service.StopHostExperiment(c.Request.Context(), name, experimentId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}