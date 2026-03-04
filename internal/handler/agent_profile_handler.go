package handler

import (
	"net/http"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AgentProfileHandler struct {
	svc       service.AgentProfileService
	tenantSvc service.TenantService
}

func NewAgentProfileHandler(svc service.AgentProfileService, tenantSvc service.TenantService) *AgentProfileHandler {
	return &AgentProfileHandler{svc: svc, tenantSvc: tenantSvc}
}

func (h *AgentProfileHandler) Create(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req service.CreateAgentProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p, err := h.svc.Create(c.Request.Context(), slug, req)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *AgentProfileHandler) List(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profiles, err := h.svc.List(c.Request.Context(), slug)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": profiles})
}

func (h *AgentProfileHandler) Update(c *gin.Context) {
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

	var req service.UpdateAgentProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	p, err := h.svc.Update(c.Request.Context(), slug, profileID, req)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, p)
}

// resolveSlug extracts tenantID from :id param and slug from JWT or DB.
func resolveSlug(c *gin.Context, tenantSvc service.TenantService) (uuid.UUID, string, error) {
	tenantID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return uuid.Nil, "", err
	}
	slug := c.GetString("tenant_slug")
	if slug == "" {
		slug, err = tenantSvc.GetSlug(c.Request.Context(), tenantID)
		if err != nil {
			return uuid.Nil, "", err
		}
	}
	return tenantID, slug, nil
}
