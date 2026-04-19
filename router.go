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

func SetupRouter() *gin.Engine {
	router := gin.Default()
	router.Use(corsMiddleware())

	authGroup := router.Group("/auth")
	{
		authGroup.POST("/login", handlers.LoginHandler)
	}

	router.POST("/validate", handlers.ValidateToken)

	apiGroup := router.Group("/api")
	apiGroup.Use(middlewares.AuthMiddleware())
	{
		adminGroup := apiGroup.Group("/admin")
		adminGroup.Use(middlewares.RoleMiddleware("app_admin"))
		{
			adminGroup.GET("/tenants", listTenantsHandler)
			adminGroup.POST("/tenants", handlers.CreateTenant)
		}

		tenantGroup := apiGroup.Group("/tenant")
		{
			tenantGroup.GET("/stats", middlewares.RoleMiddleware("tenant_admin"), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Estadísticas del tenant principal"})
			})
			tenantGroup.GET("/operations", middlewares.RoleMiddleware("tenant_admin", "tenant_operator"), func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Panel de Operaciones de este Tenant"})
			})
		}

		usersGroup := apiGroup.Group("/users")
		usersGroup.Use(middlewares.RoleMiddleware("app_admin", "tenant_admin"))
		{
			usersGroup.POST("/", handlers.CreateUser)
		}
	}

	return router
}
