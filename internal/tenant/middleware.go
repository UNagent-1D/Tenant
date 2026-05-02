package tenant

import (
	"net/http"

	"github.com/UNagent-1D/Tenant/internal/auth"
	"github.com/UNagent-1D/Tenant/pkg/db"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// TenantScopeMiddleware validates that the caller may access the tenant in :id
// and stores "tenant_slug" in context for downstream handlers.
func TenantScopeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tenantID := c.Param("id")
		claims := c.MustGet("user_claims").(*auth.Claims)

		if claims.Role != "app_admin" {
			if claims.TenantID == nil || *claims.TenantID != tenantID {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Acceso denegado: tenant incorrecto"})
				return
			}
		}

		var slug string
		err := db.Pool.QueryRow(c.Request.Context(),
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
