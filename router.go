package main

import (
	"net/http"
	"time"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/handlers"
	"github.com/UNagent-1D/Tenant/middlewares"
	"github.com/gin-gonic/gin"
)

type tenantRow struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Domain    *string   `json:"domain"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func listTenantsHandler(c *gin.Context) {
	rows, err := config.DB.Query(`SELECT id, name, domain, is_active, created_at, updated_at FROM tenants ORDER BY created_at DESC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	out := []tenantRow{}
	for rows.Next() {
		var t tenantRow
		if err := rows.Scan(&t.ID, &t.Name, &t.Domain, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		out = append(out, t)
	}
	c.JSON(http.StatusOK, out)
}

// corsMiddleware answers preflight OPTIONS and attaches the CORS headers
// every response needs so the browser-hosted frontend can call us.
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

// SetupRouter inicializa y configura todas las rutas HTTP de la aplicación
func SetupRouter() *gin.Engine {
	// 2. Levanta el enrutador de Gin
	router := gin.Default()
	router.Use(corsMiddleware())

	// 3. Rutas Públicas (No requieren Token)
	authGroup := router.Group("/auth")
	{
		// Recibe { "email": "", "password": "" } y emite un JWT firmado verificando contra SQL
		authGroup.POST("/login", handlers.LoginHandler)
	}

	// 4. API protegida general (El AuthMiddleware obliga la presencia de JWT válido)
	apiGroup := router.Group("/api")
	apiGroup.Use(middlewares.AuthMiddleware())
	{
		// 4A. Rutas del App Admin (solo "app_admin", que no tiene Tenant ID asignado, los administra todos)
		adminGroup := apiGroup.Group("/admin")
		adminGroup.Use(middlewares.RoleMiddleware("app_admin"))
		{
			adminGroup.GET("/tenants", listTenantsHandler)
		}

		// 4B. Rutas protegidas a nivel del tenant
		tenantGroup := apiGroup.Group("/tenant")
		{
			// Estadísticas: Requiere ser Tenant Admin
			tenantGroup.GET("/stats", middlewares.RoleMiddleware("tenant_admin"), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Estadísticas del tenant principal"})
			})

			// Operadores: Los tenant admins u operators pueden consultar el board de operaciones
			tenantGroup.GET("/operations", middlewares.RoleMiddleware("tenant_admin", "tenant_operator"), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Panel de Operaciones de este Tenant"})
			})
		}
	}

	return router
}
