package middlewares

import (
	"net/http"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// TenantScopeMiddleware validates that the caller may access the tenant in :id.
// It also resolves and stores "tenant_slug" in context for use by handlers.
func TenantScopeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Param("id")
		claims := c.MustGet("user_claims").(*models.Claims)

		if claims.Role != "app_admin" {
			if claims.TenantID == nil || *claims.TenantID != tenantID {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Acceso denegado: tenant incorrecto"})
				return
			}
		}

		var slug string
		err := config.DB.QueryRow(c.Request.Context(),
			"SELECT slug FROM tenants WHERE id = $1", tenantID,
		).Scan(&slug)
		if err != nil {
			if err == pgx.ErrNoRows {
				c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "Tenant no encontrado"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Error interno"})
			return
		}

		c.Set("tenant_slug", slug)
		c.Next()
	}
}
