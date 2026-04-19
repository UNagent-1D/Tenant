package main

import (
	"net/http"

	"github.com/UNagent-1D/Tenant/handlers"
	"github.com/UNagent-1D/Tenant/middlewares"
	"github.com/gin-gonic/gin"
)

// SetupRouter inicializa y configura todas las rutas HTTP de la aplicación
func SetupRouter() *gin.Engine {
	// 2. Levanta el enrutador de Gin
	router := gin.Default()

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
			adminGroup.GET("/tenants", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "Listado de todos los tenants del sistema global."})
			})

			// 4A.1. Crear un Tenant (solo app_admin)
			adminGroup.POST("/tenants", handlers.CreateTenant)

			// 4A.2. Eliminar un Tenant (solo app_admin)
			adminGroup.DELETE("/tenants/:id", handlers.DeleteTenant)
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

		// 4C. Rutas para administración de usuarios
		usersGroup := apiGroup.Group("/users")
		// Ambos, app_admin y tenant_admin, pueden crear usuarios (la lógica de qué rol crean está en el handler)
		usersGroup.Use(middlewares.RoleMiddleware("app_admin", "tenant_admin"))
		{
			usersGroup.POST("/", handlers.CreateUser)
			usersGroup.DELETE("/:id", handlers.DeleteUser)
		}
	}

	return router
}
