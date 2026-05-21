package main

import (
	"net/http"

	"github.com/UNagent-1D/Tenant/config"
	"github.com/UNagent-1D/Tenant/internal/auth"
	"github.com/UNagent-1D/Tenant/internal/tenant"
	"github.com/UNagent-1D/Tenant/middlewares"
	mw "github.com/UNagent-1D/Tenant/pkg/middleware"
	"github.com/gin-gonic/gin"
)

// tenantChain runs RoleMiddleware before TenantScopeMiddleware so that
// role enforcement (no DB) always happens before the slug resolution (needs DB).
func tenantChain(roles ...string) []gin.HandlerFunc {
	return []gin.HandlerFunc{
		auth.RoleMiddleware(roles...),
		tenant.TenantScopeMiddleware(),
	}
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
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Internal-Key")
		c.Header("Access-Control-Allow-Credentials", "true")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func SetupRouter(cfg *config.Config) *gin.Engine {
	router := gin.New()
	router.Use(corsMiddleware())
	router.Use(mw.RequestIDMiddleware())
	router.Use(mw.StructuredLoggerMiddleware())
	router.Use(gin.Recovery())

	// Public — login is rate-limited per client IP to mitigate brute force / credential stuffing.
	router.POST("/api/v1/auth/login", middlewares.LoginRateLimiter(), auth.LoginHandler(cfg))
	router.GET("/api/v1/health", HealthCheck)

	api := router.Group("/api/v1")
	api.Use(auth.AuthMiddleware(cfg))
	{
		// ── Auth (self) ───────────────────────────────────────────────────
		api.GET("/auth/me", auth.GetMe)
		api.PATCH("/auth/me/password", auth.ChangePassword)

		// ── Users ─────────────────────────────────────────────────────────
		users := api.Group("/users")
		{
			users.GET("", auth.RoleMiddleware("app_admin", "tenant_admin"), auth.GetUsers)
			users.POST("", auth.RoleMiddleware("app_admin", "tenant_admin"), auth.CreateUser)
			users.GET("/:uid", auth.RoleMiddleware("app_admin", "tenant_admin"), auth.GetUser)
			users.PATCH("/:uid", auth.RoleMiddleware("app_admin", "tenant_admin"), auth.UpdateUser)
			users.DELETE("/:uid", auth.OnlyAdminMiddleware("only app_admin can delete users"), auth.DeleteUser)
		}

		// ── Tenants ───────────────────────────────────────────────────────
		tenants := api.Group("/tenants")
		{
			tenants.GET("", auth.OnlyAdminMiddleware("only app_admin can list all tenants"), tenant.GetTenants)
			tenants.POST("", auth.OnlyAdminMiddleware("only app_admin can create tenants"), tenant.CreateTenant(cfg))

			tenantGroup := tenants.Group("/:id")
			{
				// Read access for any role inside the tenant — operators need
				// the tenant name for sidebar/branding. tenantChain still
				// enforces self-tenant scope, so an operator can only read
				// their own tenant.
				tenantGroup.GET("", append(tenantChain("tenant_admin", "tenant_operator"), tenant.GetTenant)...)
				tenantGroup.PATCH("", append(tenantChain("tenant_admin"), tenant.UpdateTenant)...)

				// Channels — tenant_admin only
				channels := tenantGroup.Group("/channels")
				channels.Use(tenantChain("tenant_admin")...)
				{
					channels.GET("", tenant.GetChannels)
					channels.POST("", tenant.CreateChannel)
					channels.PATCH("/:cid", tenant.UpdateChannel)
				}

				// Agent Profiles
				profiles := tenantGroup.Group("/profiles")
				{
					profiles.GET("", append(tenantChain("tenant_admin", "tenant_operator"), tenant.GetProfiles)...)
					profiles.POST("", append(tenantChain("tenant_admin"), tenant.CreateProfile)...)
					profiles.PATCH("/:pid", append(tenantChain("tenant_admin"), tenant.UpdateProfile)...)

					// Agent Configs (ACR)
					configs := profiles.Group("/:pid/configs")
					{
						configs.GET("", append(tenantChain("tenant_admin", "tenant_operator"), tenant.GetConfigs)...)
						configs.GET("/active", append(tenantChain("tenant_admin", "tenant_operator"), tenant.GetActiveConfig)...)
						configs.POST("", append(tenantChain("tenant_admin"), tenant.CreateConfig)...)
						configs.PATCH("/:cid", append(tenantChain("tenant_admin"), tenant.UpdateConfig)...)
						configs.POST("/:cid/activate", append(tenantChain("tenant_admin"), tenant.ActivateConfig)...)
					}
				}

				// Data Sources — tenant_admin only
				dataSources := tenantGroup.Group("/data-sources")
				dataSources.Use(tenantChain("tenant_admin")...)
				{
					dataSources.GET("", tenant.GetDataSources)
					dataSources.POST("", tenant.CreateDataSource)
					dataSources.PATCH("/:did", tenant.UpdateDataSource)
				}
			}
		}

		// ── Tool Registry ──────────────────────────────────────────────────
		tools := api.Group("/tool-registry")
		{
			tools.GET("", auth.RoleMiddleware("app_admin", "tenant_admin"), tenant.GetTools)
			tools.POST("", auth.OnlyAdminMiddleware("only app_admin can register tools"), tenant.CreateTool)
			tools.PATCH("/:tid", auth.OnlyAdminMiddleware("only app_admin can modify tools"), tenant.UpdateTool)
		}
	}

	// ── Internal (orchestrator + User-Auth) ────────────────────────────────
	internal := router.Group("/api/v1/internal")
	internal.Use(auth.InternalKeyMiddleware(cfg))
	{
		internal.POST("/tenants/:id/data-sources/:did/execute", tenant.ExecuteDataSource(cfg))
		// User-Auth posts here after verifying an OTP. Tenant upserts the
		// users row (default tenant_operator) and mints the canonical JWT.
		internal.POST("/auth/upsert-from-otp", auth.UpsertFromOTPHandler(cfg))
		// agent-runtime (and chat-orch in Phase 3) consume this to resolve
		// the active LLM model + system prompt + tools for a tenant.
		internal.GET("/tenants/:id/profiles/active", tenant.GetActiveAgentByTenantID)
	}

	return router
}
