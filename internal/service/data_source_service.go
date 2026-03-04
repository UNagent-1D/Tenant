package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/db"
	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/UNagent-1D/tenant-service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var allowedMethods = map[string]bool{
	"GET": true, "POST": true, "PATCH": true, "DELETE": true,
}

type DataSourceService interface {
	Create(ctx context.Context, slug string, req CreateDataSourceRequest) (*domain.DataSource, error)
	List(ctx context.Context, slug string) ([]*domain.DataSource, error)
	Update(ctx context.Context, slug string, dsID uuid.UUID, req UpdateDataSourceRequest) (*domain.DataSource, error)
}

type CreateDataSourceRequest struct {
	Name          string             `json:"name" binding:"required"`
	SourceType    domain.SourceType  `json:"source_type" binding:"required"`
	BaseURL       string             `json:"base_url" binding:"required"`
	CredentialRef *string            `json:"credential_ref"`
	RouteConfigs  json.RawMessage    `json:"route_configs" binding:"required"`
}

type UpdateDataSourceRequest struct {
	Name          *string         `json:"name"`
	BaseURL       *string         `json:"base_url"`
	CredentialRef *string         `json:"credential_ref"`
	RouteConfigs  json.RawMessage `json:"route_configs"`
	IsActive      *bool           `json:"is_active"`
}

type dataSourceService struct {
	repo repository.DataSourceRepository
	pool *pgxpool.Pool
}

func NewDataSourceService(repo repository.DataSourceRepository, pool *pgxpool.Pool) DataSourceService {
	return &dataSourceService{repo: repo, pool: pool}
}

func (s *dataSourceService) Create(ctx context.Context, slug string, req CreateDataSourceRequest) (*domain.DataSource, error) {
	if err := validateRouteConfigs(req.RouteConfigs); err != nil {
		return nil, err
	}

	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ds := &domain.DataSource{
		ID:            uuid.New(),
		Name:          req.Name,
		SourceType:    req.SourceType,
		BaseURL:       req.BaseURL,
		CredentialRef: req.CredentialRef,
		RouteConfigs:  req.RouteConfigs,
		IsActive:      true,
	}
	if err := s.repo.Create(ctx, conn, ds); err != nil {
		return nil, err
	}
	return ds, nil
}

func (s *dataSourceService) List(ctx context.Context, slug string) ([]*domain.DataSource, error) {
	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	return s.repo.List(ctx, conn)
}

func (s *dataSourceService) Update(ctx context.Context, slug string, dsID uuid.UUID, req UpdateDataSourceRequest) (*domain.DataSource, error) {
	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ds, err := s.repo.GetByID(ctx, conn, dsID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		ds.Name = *req.Name
	}
	if req.BaseURL != nil {
		ds.BaseURL = *req.BaseURL
	}
	if req.CredentialRef != nil {
		ds.CredentialRef = req.CredentialRef
	}
	if req.RouteConfigs != nil {
		if err := validateRouteConfigs(req.RouteConfigs); err != nil {
			return nil, err
		}
		ds.RouteConfigs = req.RouteConfigs
	}
	if req.IsActive != nil {
		ds.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, conn, ds); err != nil {
		return nil, err
	}
	return ds, nil
}

// validateRouteConfigs checks that each route entry has a valid HTTP method.
func validateRouteConfigs(raw json.RawMessage) error {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return fmt.Errorf("route_configs cannot be empty: %w", apperrors.ErrValidation)
	}

	var configs map[string]domain.RouteConfig
	if err := json.Unmarshal(raw, &configs); err != nil {
		return fmt.Errorf("route_configs must be a valid JSON object: %w", apperrors.ErrValidation)
	}

	for op, rc := range configs {
		if !allowedMethods[rc.Method] {
			return fmt.Errorf("route %q: method must be one of: GET, POST, PATCH, DELETE: %w", op, apperrors.ErrValidation)
		}
	}
	return nil
}
