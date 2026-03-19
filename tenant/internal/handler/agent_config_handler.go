package handler

import (
	"net/http"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AgentConfigHandler struct {
	svc       service.AgentConfigService
	tenantSvc service.TenantService
}

func NewAgentConfigHandler(svc service.AgentConfigService, tenantSvc service.TenantService) *AgentConfigHandler {
	return &AgentConfigHandler{svc: svc, tenantSvc: tenantSvc}
}

func (h *AgentConfigHandler) Create(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	profileID, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}
	callerID, _ := uuid.Parse(c.GetString("user_id"))

	var req service.CreateAgentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ac, err := h.svc.Create(c.Request.Context(), slug, profileID, callerID, req)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ac)
}

func (h *AgentConfigHandler) List(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	profileID, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}

	configs, err := h.svc.List(c.Request.Context(), slug, profileID)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": configs})
}

func (h *AgentConfigHandler) GetActive(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	profileID, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}

	ac, err := h.svc.GetActive(c.Request.Context(), slug, profileID)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ac)
}

func (h *AgentConfigHandler) Update(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	configID, err := uuid.Parse(c.Param("cid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config id"})
		return
	}

	var req service.UpdateAgentConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ac, err := h.svc.Update(c.Request.Context(), slug, configID, req)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ac)
}

func (h *AgentConfigHandler) Activate(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	profileID, err := uuid.Parse(c.Param("pid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid profile id"})
		return
	}
	configID, err := uuid.Parse(c.Param("cid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config id"})
		return
	}

	ac, err := h.svc.Activate(c.Request.Context(), slug, profileID, configID)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ac)
}
