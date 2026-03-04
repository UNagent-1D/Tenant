package handler

import (
	"net/http"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ToolRegistryHandler struct {
	svc service.ToolRegistryService
}

func NewToolRegistryHandler(svc service.ToolRegistryService) *ToolRegistryHandler {
	return &ToolRegistryHandler{svc: svc}
}

func (h *ToolRegistryHandler) Create(c *gin.Context) {
	var req service.CreateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (h *ToolRegistryHandler) List(c *gin.Context) {
	tools, err := h.svc.List(c.Request.Context())
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tools})
}

func (h *ToolRegistryHandler) Update(c *gin.Context) {
	tid, err := uuid.Parse(c.Param("tid"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tool id"})
		return
	}
	var req service.UpdateToolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	t, err := h.svc.Update(c.Request.Context(), tid, req)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, t)
}
