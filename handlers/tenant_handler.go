package handlers

import (
	"net/http"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
)

// CreateTenant maneja la creación de nuevos tenants (solo app_admin)
func CreateTenant(c *gin.Context) {
	var req models.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payload de solicitud inválido", "detalle": err.Error()})
		return
	}

	var tenant models.Tenant
	query := `
		INSERT INTO tenants (name, domain)
		VALUES ($1, $2)
		RETURNING id, name, domain, is_active;
	`

	// Insertamos el nuevo tenant
	var domainVal *string
	if req.Domain != "" {
		domainVal = &req.Domain
	}

	err := config.DB.QueryRow(query, req.Name, domainVal).Scan(
		&tenant.ID, &tenant.Name, &tenant.Domain, &tenant.IsActive,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al crear el tenant", "detalle": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tenant creado exitosamente",
		"tenant":  tenant,
	})
}
