package main

import (
	"github.com/UNagent-1D/Tenant/handlers"
	"github.com/UNagent-1D/Tenant/middlewares"
	"github.com/gin-gonic/gin"
)

// role runs RoleMiddleware followed by TenantScopeMiddleware so that
// role enforcement (no DB) always runs before the scope resolution (needs DB).
func tenantChain(roles ...string) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middlewares.RoleMiddleware(roles...),
		middlewares.TenantScopeMiddleware(),
	}
}

func SetupRouter() *gin.Engine {
	router := gin.New()
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
				// Tenant resource itself
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
