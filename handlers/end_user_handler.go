package handlers

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

func CreateEndUser(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)
	tenantID := c.Param("id")

	var req models.CreateEndUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var eu models.EndUser
	err := config.DB.QueryRow(c.Request.Context(),
		fmt.Sprintf(`INSERT INTO %s.end_users (tenant_id, full_name, national_id, cellphone, email, date_of_birth, external_ref)
		             VALUES ($1, $2, $3, $4, $5, $6, $7)
		             RETURNING id, tenant_id, full_name, national_id, cellphone, email, is_active, external_ref, created_at`, schema),
		tenantID, req.FullName, req.NationalID, req.Cellphone, req.Email, req.DateOfBirth, req.ExternalRef,
	).Scan(&eu.ID, &eu.TenantID, &eu.FullName, &eu.NationalID, &eu.Cellphone,
		&eu.Email, &eu.IsActive, &eu.ExternalRef, &eu.CreatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "national_id o cellphone ya registrado para este tenant"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al registrar usuario final"})
		return
	}
	c.JSON(http.StatusCreated, eu)
}

// LookupByPhone always returns 200 to prevent enumeration attacks.
func LookupByPhone(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)
	tenantID := c.Param("id")

	var id string
	err := config.DB.QueryRow(c.Request.Context(),
		fmt.Sprintf(`SELECT id FROM %s.end_users WHERE tenant_id = $1 AND cellphone = $2 AND is_active = true`, schema),
		tenantID, c.Param("number"),
	).Scan(&id)

	if err != nil {
		c.JSON(http.StatusOK, models.LookupResponse{Exists: false})
		return
	}
	c.JSON(http.StatusOK, models.LookupResponse{Exists: true, UserID: &id})
}

// LookupByNationalID always returns 200 to prevent enumeration attacks.
func LookupByNationalID(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)
	tenantID := c.Param("id")

	var id string
	err := config.DB.QueryRow(c.Request.Context(),
		fmt.Sprintf(`SELECT id FROM %s.end_users WHERE tenant_id = $1 AND national_id = $2 AND is_active = true`, schema),
		tenantID, c.Param("nid"),
	).Scan(&id)

	if err != nil {
		c.JSON(http.StatusOK, models.LookupResponse{Exists: false})
		return
	}
	c.JSON(http.StatusOK, models.LookupResponse{Exists: true, UserID: &id})
}
