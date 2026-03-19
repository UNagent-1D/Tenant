package handler

import (
	"net/http"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ChannelHandler struct {
	svc        service.ChannelService
	tenantSvc  service.TenantService
}

func NewChannelHandler(svc service.ChannelService, tenantSvc service.TenantService) *ChannelHandler {
	return &ChannelHandler{svc: svc, tenantSvc: tenantSvc}
}

func (h *ChannelHandler) Create(c *gin.Context) {
	tenantID, slug, err := h.resolveTenant(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req service.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ch, err := h.svc.Create(c.Request.Context(), tenantID, slug, req)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ch)
}

func (h *ChannelHandler) List(c *gin.Context) {
	tenantID, slug, err := h.resolveTenant(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channels, err := h.svc.List(c.Request.Context(), tenantID, slug)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": channels})
}

func (h *ChannelHandler) Update(c *gin.Context) {
	tenantID, slug, err := h.resolveTenant(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	channelID, err := uuid.Parse(c.Param("cid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid channel id"})
		return
	}

	var req service.UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ch, err := h.svc.Update(c.Request.Context(), tenantID, slug, channelID, req)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ch)
}

// resolveTenant extracts the tenant UUID and slug from the request.
// If the JWT already has a slug (tenant_admin), use it directly to avoid a DB lookup.
func (h *ChannelHandler) resolveTenant(c *gin.Context) (uuid.UUID, string, error) {
	idStr := c.Param("id")
	tenantID, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil, "", err
	}

	slug := c.GetString("tenant_slug")
	if slug == "" {
		// app_admin: resolve slug from DB
		slug, err = h.tenantSvc.GetSlug(c.Request.Context(), tenantID)
		if err != nil {
			return uuid.Nil, "", err
		}
	}
	return tenantID, slug, nil
}
