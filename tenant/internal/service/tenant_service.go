package service

import (
	"context"
	"fmt"
	"regexp"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/db"
	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/UNagent-1D/tenant-service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type TenantService interface {
	Create(ctx context.Context, req CreateTenantRequest) (*domain.Tenant, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	List(ctx context.Context, page, limit int) ([]*domain.Tenant, int, error)
	Update(ctx context.Context, id uuid.UUID, req UpdateTenantRequest) (*domain.Tenant, error)
	GetSlug(ctx context.Context, tenantID uuid.UUID) (string, error)
}

type CreateTenantRequest struct {
	Name                 string           `json:"name" binding:"required"`
	Slug                 string           `json:"slug" binding:"required"`
	Plan                 domain.TenantPlan `json:"plan" binding:"required"`
	BrandingLogoURL      *string          `json:"branding_logo_url"`
	BrandingPrimaryColor *string          `json:"branding_primary_color"`
}

type UpdateTenantRequest struct {
	Name                 *string           `json:"name"`
	Plan                 *domain.TenantPlan `json:"plan"`
	Status               *domain.TenantStatus `json:"status"`
	BrandingLogoURL      *string           `json:"branding_logo_url"`
	BrandingPrimaryColor *string           `json:"branding_primary_color"`
}

type tenantService struct {
	repo             repository.TenantRepository
	pool             *pgxpool.Pool
	baseDSN          string
	tenantMigrations string
}

func NewTenantService(repo repository.TenantRepository, pool *pgxpool.Pool, baseDSN, tenantMigrationsPath string) TenantService {
	return &tenantService{
		repo:             repo,
		pool:             pool,
		baseDSN:          baseDSN,
		tenantMigrations: tenantMigrationsPath,
	}
}

func (s *tenantService) Create(ctx context.Context, req CreateTenantRequest) (*domain.Tenant, error) {
	if !slugPattern.MatchString(req.Slug) {
		return nil, fmt.Errorf("slug must be lowercase alphanumeric with hyphens only: %w", apperrors.ErrValidation)
	}
	if db.IsReservedSlug(req.Slug) {
		return nil, fmt.Errorf("slug %q is reserved: %w", req.Slug, apperrors.ErrValidation)
	}
	if !isValidPlan(req.Plan) {
		return nil, fmt.Errorf("plan must be one of: free, starter, pro, enterprise: %w", apperrors.ErrValidation)
	}

	exists, err := s.repo.SlugExists(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("a tenant with slug %q already exists: %w", req.Slug, apperrors.ErrConflict)
	}

	t := &domain.Tenant{
		ID:                   uuid.New(),
		Slug:                 req.Slug,
		Name:                 req.Name,
		Plan:                 req.Plan,
		Status:               domain.StatusActive,
		BrandingLogoURL:      req.BrandingLogoURL,
		BrandingPrimaryColor: req.BrandingPrimaryColor,
	}

	if err := s.repo.Create(ctx, t); err != nil {
		return nil, err
	}

	// Provision per-tenant schema + run tenant migrations
	if err := db.ProvisionTenantSchema(ctx, s.pool, t.Slug, s.baseDSN); err != nil {
		// Compensate: remove the tenant row
		_ = s.repo.Update(ctx, &domain.Tenant{ID: t.ID, Status: domain.StatusChurned})
		return nil, fmt.Errorf("provision tenant schema: %w", err)
	}

	return t, nil
}

func (s *tenantService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *tenantService) List(ctx context.Context, page, limit int) ([]*domain.Tenant, int, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	return s.repo.List(ctx, page, limit)
}

func (s *tenantService) Update(ctx context.Context, id uuid.UUID, req UpdateTenantRequest) (*domain.Tenant, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Plan != nil {
		if !isValidPlan(*req.Plan) {
			return nil, fmt.Errorf("plan must be one of: free, starter, pro, enterprise: %w", apperrors.ErrValidation)
		}
		t.Plan = *req.Plan
	}
	if req.Status != nil {
		if !isValidStatus(*req.Status) {
			return nil, fmt.Errorf("status must be one of: active, suspended, churned: %w", apperrors.ErrValidation)
		}
		t.Status = *req.Status
	}
	if req.BrandingLogoURL != nil {
		t.BrandingLogoURL = req.BrandingLogoURL
	}
	if req.BrandingPrimaryColor != nil {
		t.BrandingPrimaryColor = req.BrandingPrimaryColor
	}
	if err := s.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

func (s *tenantService) GetSlug(ctx context.Context, tenantID uuid.UUID) (string, error) {
	t, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return "", err
	}
	return t.Slug, nil
}

func isValidPlan(p domain.TenantPlan) bool {
	switch p {
	case domain.PlanFree, domain.PlanStarter, domain.PlanPro, domain.PlanEnterprise:
		return true
	}
	return false
}

func isValidStatus(s domain.TenantStatus) bool {
	switch s {
	case domain.StatusActive, domain.StatusSuspended, domain.StatusChurned:
		return true
	}
	return false
}
