package handler

import (
	"net/http"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserHandler struct {
	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

func (h *UserHandler) Create(c *gin.Context) {
	var req service.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	callerRole := c.GetString("role")
	callerTenantID := parseTenantIDFromCtx(c)

	u, err := h.svc.Create(c.Request.Context(), req, callerRole, callerTenantID)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, u)
}

func (h *UserHandler) List(c *gin.Context) {
	callerRole := c.GetString("role")
	callerTenantID := parseTenantIDFromCtx(c)

	var tenantFilter *uuid.UUID
	if callerRole == "tenant_admin" {
		tenantFilter = callerTenantID
	} else if raw := c.Query("tenant_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err == nil {
			tenantFilter = &id
		}
	}

	page := queryInt(c, "page", 1)
	limit := queryInt(c, "limit", 20)

	users, total, err := h.svc.List(c.Request.Context(), tenantFilter, page, limit)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data":       users,
		"pagination": gin.H{"page": page, "limit": limit, "total": total},
	})
}

func (h *UserHandler) Update(c *gin.Context) {
	uid, err := uuid.Parse(c.Param("uid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	var req service.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	callerID, _ := uuid.Parse(c.GetString("user_id"))
	callerRole := c.GetString("role")
	callerTenantID := parseTenantIDFromCtx(c)

	u, err := h.svc.Update(c.Request.Context(), uid, req, callerID, callerRole, callerTenantID)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, u)
}

func parseTenantIDFromCtx(c *gin.Context) *uuid.UUID {
	raw := c.GetString("tenant_id")
	if raw == "" {
		return nil
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return nil
	}
	return &id
}
