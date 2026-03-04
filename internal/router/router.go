package router

import (
	"log/slog"

	"github.com/UNagent-1D/tenant-service/internal/handler"
	"github.com/UNagent-1D/tenant-service/internal/middleware"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Health       *handler.HealthHandler
	Tenant       *handler.TenantHandler
	User         *handler.UserHandler
	Channel      *handler.ChannelHandler
	AgentProfile *handler.AgentProfileHandler
	DataSource   *handler.DataSourceHandler
	EndUser      *handler.EndUserHandler
	Tool         *handler.ToolRegistryHandler
	AgentConfig  *handler.AgentConfigHandler
}

func New(ginMode string, authCfg middleware.AuthConfig, logger *slog.Logger, h Handlers) *gin.Engine {
	gin.SetMode(ginMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Health check — no auth, no logging overhead
	r.GET("/api/v1/health", h.Health.Check)

	// Base API group: RequestID + Logger on everything
	api := r.Group("/api/v1")
	api.Use(middleware.RequestID(), middleware.Logger(logger))

	// Public: auth login (owned by the auth microservice — not implemented here)
	// If you proxy it, uncomment:
	// api.POST("/auth/login", proxyToAuthService)

	// Authenticated routes
	auth := api.Group("")
	auth.Use(middleware.Auth(authCfg))

	// --- Tenants ---
	tenants := auth.Group("/tenants")
	tenants.POST("", middleware.RequireRole("app_admin"), h.Tenant.Create)
	tenants.GET("", middleware.RequireRole("app_admin"), h.Tenant.List)
	tenants.GET("/:id", middleware.RequireTenantAccess(), h.Tenant.GetByID)
	tenants.PATCH("/:id", middleware.RequireTenantAccess(), h.Tenant.Update)

	// Channels
	tenants.GET("/:id/channels", middleware.RequireTenantAccess(), middleware.RequireRole("app_admin", "tenant_admin"), h.Channel.List)
	tenants.POST("/:id/channels", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin"), h.Channel.Create)
	tenants.PATCH("/:id/channels/:cid", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin"), h.Channel.Update)

	// Agent Profiles
	tenants.GET("/:id/profiles", middleware.RequireTenantAccess(), middleware.RequireRole("app_admin", "tenant_admin", "tenant_operator"), h.AgentProfile.List)
	tenants.POST("/:id/profiles", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin"), h.AgentProfile.Create)
	tenants.PATCH("/:id/profiles/:pid", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin"), h.AgentProfile.Update)

	// Agent Configs (ACR)
	tenants.GET("/:id/profiles/:pid/configs", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin", "tenant_operator"), h.AgentConfig.List)
	tenants.GET("/:id/profiles/:pid/configs/active", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin", "tenant_operator"), h.AgentConfig.GetActive)
	tenants.POST("/:id/profiles/:pid/configs", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin"), h.AgentConfig.Create)
	tenants.PATCH("/:id/profiles/:pid/configs/:cid", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin"), h.AgentConfig.Update)
	tenants.POST("/:id/profiles/:pid/configs/:cid/activate", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin"), h.AgentConfig.Activate)

	// Data Sources
	tenants.GET("/:id/data-sources", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin"), h.DataSource.List)
	tenants.POST("/:id/data-sources", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin"), h.DataSource.Create)
	tenants.PATCH("/:id/data-sources/:did", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin"), h.DataSource.Update)

	// End Users
	tenants.GET("/:id/end-users/lookup/phone/:number", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin", "tenant_operator"), h.EndUser.LookupByPhone)
	tenants.GET("/:id/end-users/lookup/national-id/:nid", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin", "tenant_operator"), h.EndUser.LookupByNationalID)
	tenants.POST("/:id/end-users", middleware.RequireTenantAccess(), middleware.RequireRole("tenant_admin"), h.EndUser.Register)

	// --- Users ---
	auth.GET("/users", middleware.RequireRole("app_admin", "tenant_admin"), h.User.List)
	auth.POST("/users", middleware.RequireRole("app_admin", "tenant_admin"), h.User.Create)
	auth.PATCH("/users/:uid", middleware.RequireRole("app_admin", "tenant_admin"), h.User.Update)

	// --- Tool Registry (ACR — global) ---
	auth.GET("/tool-registry", middleware.RequireRole("app_admin", "tenant_admin"), h.Tool.List)
	auth.POST("/tool-registry", middleware.RequireRole("app_admin"), h.Tool.Create)
	auth.PATCH("/tool-registry/:tid", middleware.RequireRole("app_admin"), h.Tool.Update)

	return r
}
