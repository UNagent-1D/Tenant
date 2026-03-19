package handler

import (
	"net/http"
	"net/url"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/UNagent-1D/tenant-service/internal/service"
	"github.com/gin-gonic/gin"
)

type EndUserHandler struct {
	svc       service.EndUserService
	tenantSvc service.TenantService
}

func NewEndUserHandler(svc service.EndUserService, tenantSvc service.TenantService) *EndUserHandler {
	return &EndUserHandler{svc: svc, tenantSvc: tenantSvc}
}

// LookupByPhone handles GET /tenants/:id/end-users/lookup/phone/:number
// Always returns 200 to prevent enumeration attacks.
func (h *EndUserHandler) LookupByPhone(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// URL-decode the phone number (%2B → +)
	number, _ := url.PathUnescape(c.Param("number"))

	eu, err := h.svc.LookupByPhone(c.Request.Context(), slug, number)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, lookupResponse(eu))
}

// LookupByNationalID handles GET /tenants/:id/end-users/lookup/national-id/:nid
// Always returns 200 to prevent enumeration attacks.
func (h *EndUserHandler) LookupByNationalID(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	nid := c.Param("nid")
	eu, err := h.svc.LookupByNationalID(c.Request.Context(), slug, nid)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, lookupResponse(eu))
}

// Register handles POST /tenants/:id/end-users
func (h *EndUserHandler) Register(c *gin.Context) {
	_, slug, err := resolveSlug(c, h.tenantSvc)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var req service.RegisterEndUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	eu, err := h.svc.Register(c.Request.Context(), slug, req)
	if err != nil {
		c.JSON(apperrors.HTTPStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, eu)
}

// lookupResponse builds the anti-enumeration response: always 200.
func lookupResponse(eu *domain.EndUser) gin.H {
	if eu == nil {
		return gin.H{"exists": false}
	}
	return gin.H{"exists": true, "patient": eu}
}
