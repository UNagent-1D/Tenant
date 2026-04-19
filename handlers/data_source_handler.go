package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

func GetDataSources(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	rows, err := config.DB.Query(c.Request.Context(),
		fmt.Sprintf(`SELECT id, name, source_type, base_url, credential_ref, route_configs, is_active
		             FROM %s.data_sources`, schema),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al listar data sources"})
		return
	}
	defer rows.Close()

	sources := []models.DataSource{}
	for rows.Next() {
		var ds models.DataSource
		if err := rows.Scan(&ds.ID, &ds.Name, &ds.SourceType, &ds.BaseURL,
			&ds.CredentialRef, &ds.RouteConfigs, &ds.IsActive); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al leer data sources"})
			return
		}
		sources = append(sources, ds)
	}
	c.JSON(http.StatusOK, sources)
}

func CreateDataSource(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var req models.CreateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var ds models.DataSource
	err := config.DB.QueryRow(c.Request.Context(),
		fmt.Sprintf(`INSERT INTO %s.data_sources (name, source_type, base_url, credential_ref, route_configs)
		             VALUES ($1, $2, $3, $4, $5)
		             RETURNING id, name, source_type, base_url, credential_ref, route_configs, is_active`, schema),
		req.Name, req.SourceType, req.BaseURL, req.CredentialRef, req.RouteConfigs,
	).Scan(&ds.ID, &ds.Name, &ds.SourceType, &ds.BaseURL, &ds.CredentialRef, &ds.RouteConfigs, &ds.IsActive)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear data source"})
		return
	}
	c.JSON(http.StatusCreated, ds)
}

func UpdateDataSource(c *gin.Context) {
	slug := c.MustGet("tenant_slug").(string)
	schema := fmt.Sprintf("tenant_%s", slug)

	var req models.UpdateDataSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	setClauses := []string{}
	args := []any{}
	i := 1

	if req.RouteConfigs != nil {
		setClauses = append(setClauses, fmt.Sprintf("route_configs = $%d", i))
		args = append(args, req.RouteConfigs)
		i++
	}
	if req.CredentialRef != nil {
		setClauses = append(setClauses, fmt.Sprintf("credential_ref = $%d", i))
		args = append(args, *req.CredentialRef)
		i++
	}
	if req.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", i))
		args = append(args, *req.IsActive)
		i++
	}
	if len(setClauses) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sin campos para actualizar"})
		return
	}
	args = append(args, c.Param("did"))

	query := fmt.Sprintf(
		`UPDATE %s.data_sources SET %s WHERE id = $%d
		 RETURNING id, name, source_type, base_url, credential_ref, route_configs, is_active`,
		schema, strings.Join(setClauses, ", "), i,
	)

	var ds models.DataSource
	err := config.DB.QueryRow(c.Request.Context(), query, args...).Scan(
		&ds.ID, &ds.Name, &ds.SourceType, &ds.BaseURL, &ds.CredentialRef, &ds.RouteConfigs, &ds.IsActive,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data source no encontrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al actualizar data source"})
		return
	}
	c.JSON(http.StatusOK, ds)
}
