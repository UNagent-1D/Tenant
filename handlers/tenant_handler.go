package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func GetTenants(c *gin.Context) {
	rows, err := config.DB.Query(c.Request.Context(),
		`SELECT id, slug, name, plan, status, branding_logo_url, branding_primary_color, created_at, updated_at
		 FROM tenants ORDER BY created_at DESC`,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar tenants"})
		return
	}
	defer rows.Close()

	tenants := []models.Tenant{}
	for rows.Next() {
		var t models.Tenant
		if err := rows.Scan(&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status,
			&t.BrandingLogoURL, &t.BrandingPrimaryColor, &t.CreatedAt, &t.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer tenants"})
			return
		}
		tenants = append(tenants, t)
	}
	c.JSON(http.StatusOK, tenants)
}

func GetTenant(c *gin.Context) {
	var t models.Tenant
	err := config.DB.QueryRow(c.Request.Context(),
		`SELECT id, slug, name, plan, status, branding_logo_url, branding_primary_color, created_at, updated_at
		 FROM tenants WHERE id = $1`,
		c.Param("id"),
	).Scan(&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status,
		&t.BrandingLogoURL, &t.BrandingPrimaryColor, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant no encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error interno"})
		return
	}
	c.JSON(http.StatusOK, t)
}

func CreateTenant(c *gin.Context) {
	var req models.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx, err := config.DB.Begin(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al iniciar transacción"})
		return
	}
	defer tx.Rollback(c.Request.Context())

	var t models.Tenant
	err = tx.QueryRow(c.Request.Context(),
		`INSERT INTO tenants (slug, name, plan, branding_logo_url, branding_primary_color)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, slug, name, plan, status, branding_logo_url, branding_primary_color, created_at, updated_at`,
		req.Slug, req.Name, req.Plan, req.BrandingLogoURL, req.BrandingPrimaryColor,
	).Scan(&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status,
		&t.BrandingLogoURL, &t.BrandingPrimaryColor, &t.CreatedAt, &t.UpdatedAt)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": "El slug ya existe"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear tenant"})
		return
	}

	if _, err = tx.Exec(c.Request.Context(), "SELECT provision_tenant_schema($1)", t.Slug); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al provisionar schema del tenant"})
		return
	}

	if err = tx.Commit(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al confirmar transacción"})
		return
	}

	c.JSON(http.StatusCreated, t)
}

func UpdateTenant(c *gin.Context) {
	var req models.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	setClauses := []string{}
	args := []any{}
	i := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", i))
		args = append(args, *req.Name)
		i++
	}
	if req.Plan != nil {
		setClauses = append(setClauses, fmt.Sprintf("plan = $%d", i))
		args = append(args, *req.Plan)
		i++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", i))
		args = append(args, *req.Status)
		i++
	}
	if req.BrandingLogoURL != nil {
		setClauses = append(setClauses, fmt.Sprintf("branding_logo_url = $%d", i))
		args = append(args, *req.BrandingLogoURL)
		i++
	}
	if req.BrandingPrimaryColor != nil {
		setClauses = append(setClauses, fmt.Sprintf("branding_primary_color = $%d", i))
		args = append(args, *req.BrandingPrimaryColor)
		i++
	}

	if len(setClauses) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sin campos para actualizar"})
		return
	}
	setClauses = append(setClauses, "updated_at = NOW()")
	args = append(args, c.Param("id"))

	query := fmt.Sprintf(
		`UPDATE tenants SET %s WHERE id = $%d
		 RETURNING id, slug, name, plan, status, branding_logo_url, branding_primary_color, created_at, updated_at`,
		strings.Join(setClauses, ", "), i,
	)

	var t models.Tenant
	err := config.DB.QueryRow(c.Request.Context(), query, args...).Scan(
		&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status,
		&t.BrandingLogoURL, &t.BrandingPrimaryColor, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Tenant no encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar tenant"})
		return
	}
	c.JSON(http.StatusOK, t)
}
