package handler

import (
	"net/http"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type DataSourceHandler struct {
	svc       service.DataSourceService
	tenantSvc service.TenantService
}

func NewDataSourceHandler(svc service.DataSourceService, tenantSvc service.TenantService) *DataSourceHandler {
	return &DataSourceHandler{svc: svc, tenantSvc: tenantSvc}
}

func (h *DataSourceHandler) Create(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req service.CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ds, err := h.svc.Create(c.Request.Context(), slug, req)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, ds)
}

func (h *DataSourceHandler) List(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sources, err := h.svc.List(c.Request.Context(), slug)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": sources})
}

func (h *DataSourceHandler) Update(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dsID, err := uuid.Parse(c.Param("did"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data source id"})
		return
	}

	var req service.UpdateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ds, err := h.svc.Update(c.Request.Context(), slug, dsID, req)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ds)
}
