package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/UNagent-1D/tenant-service/internal/config"
	"github.com/UNagent-1D/tenant-service/internal/db"
	"github.com/UNagent-1D/tenant-service/internal/handler"
	"github.com/UNagent-1D/tenant-service/internal/middleware"
	"github.com/UNagent-1D/tenant-service/internal/repository"
	"github.com/UNagent-1D/tenant-service/internal/router"
	"github.com/UNagent-1D/tenant-service/internal/service"
)

func main() {
	cfg := config.Load()

	// Structured JSON logger (outputs to stdout)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	ctx := context.Background()

	// --- Database connection pool ---
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	// Apply global migrations at startup
	if err := db.RunGlobalMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("global migrations: %v", err)
	}
	logger.Info("global migrations applied")

	// --- Global schema repositories ---
	tenantRepo := repository.NewTenantRepository(pool)
	userRepo   := repository.NewUserRepository(pool)
	toolRepo   := repository.NewToolRepository(pool)

	// --- Tenant-scoped repositories (stateless — connection injected per call) ---
	channelRepo     := repository.NewChannelRepository()
	profRepo        := repository.NewAgentProfileRepository()
	dsRepo          := repository.NewDataSourceRepository()
	agentConfigRepo := repository.NewAgentConfigRepository()
	endUserRepo     := repository.NewEndUserRepository()

	// --- Services ---
	tenantSvc      := service.NewTenantService(tenantRepo, pool, cfg.DatabaseURL, "migrations/tenant")
	userSvc        := service.NewUserService(userRepo)
	channelSvc     := service.NewChannelService(channelRepo, pool)
	profileSvc     := service.NewAgentProfileService(profRepo, pool)
	dsSvc          := service.NewDataSourceService(dsRepo, pool)
	endUserSvc     := service.NewEndUserService(endUserRepo, pool)
	toolSvc        := service.NewToolRegistryService(toolRepo)
	agentConfigSvc := service.NewAgentConfigService(agentConfigRepo, profRepo, toolRepo, pool)

	// --- Handlers ---
	h := router.Handlers{
		Health:       handler.NewHealthHandler(pool, cfg.AppVersion),
		Tenant:       handler.NewTenantHandler(tenantSvc),
		User:         handler.NewUserHandler(userSvc),
		Channel:      handler.NewChannelHandler(channelSvc, tenantSvc),
		AgentProfile: handler.NewAgentProfileHandler(profileSvc, tenantSvc),
		DataSource:   handler.NewDataSourceHandler(dsSvc, tenantSvc),
		EndUser:      handler.NewEndUserHandler(endUserSvc, tenantSvc),
		Tool:         handler.NewToolRegistryHandler(toolSvc),
		AgentConfig:  handler.NewAgentConfigHandler(agentConfigSvc, tenantSvc),
	}

	// --- Auth middleware config ---
	// When AUTH_STUB=true, the middleware injects stub claims instead of calling the auth service.
	stubTenantID := cfg.AuthStubClaims.TenantID
	stubSlug := cfg.AuthStubClaims.TenantSlug

	var stubTenantIDPtr, stubSlugPtr *string
	if stubTenantID != "" {
		stubTenantIDPtr = &stubTenantID
	}
	if stubSlug != "" {
		stubSlugPtr = &stubSlug
	}

	authCfg := middleware.AuthConfig{
		ServiceURL: cfg.AuthServiceURL,
		Stub:       cfg.AuthStub,
		StubClaims: middleware.Claims{
			UserID:     cfg.AuthStubClaims.UserID,
			Role:       cfg.AuthStubClaims.Role,
			TenantID:   stubTenantIDPtr,
			TenantSlug: stubSlugPtr,
			Email:      cfg.AuthStubClaims.Email,
		},
	}

	// --- Start server ---
	r := router.New(cfg.GinMode, authCfg, logger, h)

	addr := ":" + cfg.ServerPort
	logger.Info("server starting", slog.String("addr", addr), slog.String("version", cfg.AppVersion))
	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
