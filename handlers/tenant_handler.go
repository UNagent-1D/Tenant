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

// DeleteTenant maneja la eliminación de tenants y usuarios asociados (solo app_admin)
func DeleteTenant(c *gin.Context) {
	tenantID := c.Param("id")

	// Iniciar transacción
	tx, err := config.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al iniciar la transacción"})
		return
	}
	defer tx.Rollback()

	// Eliminar usuarios asociados al tenant primero
	_, err = tx.Exec("DELETE FROM users WHERE id IN (SELECT user_id FROM user_tenants WHERE tenant_id = $1)", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar los usuarios del tenant", "detalle": err.Error()})
		return
	}

	// Eliminar el tenant
	res, err := tx.Exec("DELETE FROM tenants WHERE id = $1", tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al eliminar el tenant", "detalle": err.Error()})
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant no encontrado"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error al finalizar la transacción"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tenant y sus usuarios eliminados exitosamente"})
}
