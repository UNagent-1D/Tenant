package main

import (
	"net/http"

	"github.com/UNagent-1D/Tenant/handlers"
	"github.com/UNagent-1D/Tenant/middlewares"
	"github.com/gin-gonic/gin"
)

// tenantChain runs RoleMiddleware before TenantScopeMiddleware so that
// role enforcement (no DB) always happens before the slug resolution (needs DB).
func tenantChain(roles ...string) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middlewares.RoleMiddleware(roles...),
		middlewares.TenantScopeMiddleware(),
	}
}

// corsMiddleware sets CORS headers and handles preflight OPTIONS requests.
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
	router := gin.New()
	router.Use(corsMiddleware())
	router.Use(middlewares.RequestIDMiddleware())
	router.Use(middlewares.StructuredLoggerMiddleware())
	router.Use(gin.Recovery())

	// Public
	router.POST("/auth/login", handlers.LoginHandler)
	router.GET("/health", HealthCheck)

	api := router.Group("/api/v1")
	api.Use(middlewares.AuthMiddleware())
	{
		// ── Users ────────────────────────────────────────────────────────
		users := api.Group("/users")
		{
			users.GET("", middlewares.RoleMiddleware("app_admin"), handlers.GetUsers)
			users.POST("", middlewares.RoleMiddleware("app_admin", "tenant_admin"), handlers.CreateUser)
			users.PATCH("/:uid", middlewares.RoleMiddleware("app_admin", "tenant_admin"), handlers.UpdateUser)
		}

		// ── Tenants ──────────────────────────────────────────────────────
		tenants := api.Group("/tenants")
		{
			tenants.GET("", middlewares.RoleMiddleware("app_admin"), handlers.GetTenants)
			tenants.POST("", middlewares.RoleMiddleware("app_admin"), handlers.CreateTenant)

			tenant := tenants.Group("/:id")
			{
				tenant.GET("", append(tenantChain("tenant_admin"), handlers.GetTenant)...)
				tenant.PATCH("", append(tenantChain("tenant_admin"), handlers.UpdateTenant)...)

				// Channels — tenant_admin only
				channels := tenant.Group("/channels")
				channels.Use(tenantChain("tenant_admin")...)
				{
					channels.GET("", handlers.GetChannels)
					channels.POST("", handlers.CreateChannel)
					channels.PATCH("/:cid", handlers.UpdateChannel)
				}

				// Agent Profiles
				profiles := tenant.Group("/profiles")
				{
					profiles.GET("", append(tenantChain("tenant_admin", "tenant_operator"), handlers.GetProfiles)...)
					profiles.POST("", append(tenantChain("tenant_admin"), handlers.CreateProfile)...)
					profiles.PATCH("/:pid", append(tenantChain("tenant_admin"), handlers.UpdateProfile)...)

					// Agent Configs (ACR)
					configs := profiles.Group("/:pid/configs")
					{
						configs.GET("", append(tenantChain("tenant_admin", "tenant_operator"), handlers.GetConfigs)...)
						configs.GET("/active", append(tenantChain("tenant_admin", "tenant_operator"), handlers.GetActiveConfig)...)
						configs.POST("", append(tenantChain("tenant_admin"), handlers.CreateConfig)...)
						configs.PATCH("/:cid", append(tenantChain("tenant_admin"), handlers.UpdateConfig)...)
						configs.POST("/:cid/activate", append(tenantChain("tenant_admin"), handlers.ActivateConfig)...)
					}
				}

				// Data Sources — tenant_admin only
				dataSources := tenant.Group("/data-sources")
				dataSources.Use(tenantChain("tenant_admin")...)
				{
					dataSources.GET("", handlers.GetDataSources)
					dataSources.POST("", handlers.CreateDataSource)
					dataSources.PATCH("/:did", handlers.UpdateDataSource)
				}

				// End Users
				endUsers := tenant.Group("/end-users")
				{
					endUsers.POST("", append(tenantChain("tenant_admin"), handlers.CreateEndUser)...)
					endUsers.GET("/lookup/phone/:number", append(tenantChain("tenant_admin", "tenant_operator"), handlers.LookupByPhone)...)
					endUsers.GET("/lookup/national-id/:nid", append(tenantChain("tenant_admin", "tenant_operator"), handlers.LookupByNationalID)...)
				}
			}
		}

		// ── Tool Registry ─────────────────────────────────────────────────
		tools := api.Group("/tool-registry")
		{
			tools.GET("", middlewares.RoleMiddleware("app_admin", "tenant_admin"), handlers.GetTools)
			tools.POST("", middlewares.RoleMiddleware("app_admin"), handlers.CreateTool)
			tools.PATCH("/:tid", middlewares.RoleMiddleware("app_admin"), handlers.UpdateTool)
		}
	}

	return router
}
