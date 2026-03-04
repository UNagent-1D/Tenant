package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgTenantRepo struct {
	db *pgxpool.Pool
}

func NewTenantRepository(db *pgxpool.Pool) TenantRepository {
	return &pgTenantRepo{db: db}
}

func (r *pgTenantRepo) Create(ctx context.Context, t *domain.Tenant) error {
	q := `
		INSERT INTO tenants (id, slug, name, plan, status, branding_logo_url, branding_primary_color)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, q,
		t.ID, t.Slug, t.Name, t.Plan, t.Status,
		t.BrandingLogoURL, t.BrandingPrimaryColor,
	).Scan(&t.CreatedAt, &t.UpdatedAt)
}

func (r *pgTenantRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	q := `SELECT id, slug, name, plan, status, branding_logo_url, branding_primary_color, created_at, updated_at
		  FROM tenants WHERE id = $1`
	t, err := scanTenant(r.db.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("tenant %s: %w", id, apperrors.ErrNotFound)
	}
	return t, err
}

func (r *pgTenantRepo) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	q := `SELECT id, slug, name, plan, status, branding_logo_url, branding_primary_color, created_at, updated_at
		  FROM tenants WHERE slug = $1`
	t, err := scanTenant(r.db.QueryRow(ctx, q, slug))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("tenant %q: %w", slug, apperrors.ErrNotFound)
	}
	return t, err
}

func (r *pgTenantRepo) List(ctx context.Context, page, limit int) ([]*domain.Tenant, int, error) {
	offset := (page - 1) * limit

	var total int
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM tenants`).Scan(&total); err != nil {
		return nil, 0, err
	}

	q := `SELECT id, slug, name, plan, status, branding_logo_url, branding_primary_color, created_at, updated_at
		  FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	rows, err := r.db.Query(ctx, q, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tenants []*domain.Tenant
	for rows.Next() {
		t, err := scanTenantRow(rows)
		if err != nil {
			return nil, 0, err
		}
		tenants = append(tenants, t)
	}
	return tenants, total, rows.Err()
}

func (r *pgTenantRepo) Update(ctx context.Context, t *domain.Tenant) error {
	q := `UPDATE tenants
		  SET name = $2, plan = $3, status = $4,
		      branding_logo_url = $5, branding_primary_color = $6,
		      updated_at = NOW()
		  WHERE id = $1
		  RETURNING updated_at`
	err := r.db.QueryRow(ctx, q,
		t.ID, t.Name, t.Plan, t.Status,
		t.BrandingLogoURL, t.BrandingPrimaryColor,
	).Scan(&t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("tenant %s: %w", t.ID, apperrors.ErrNotFound)
	}
	return err
}

func (r *pgTenantRepo) SlugExists(ctx context.Context, slug string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)`, slug).Scan(&exists)
	return exists, err
}

func scanTenant(row pgx.Row) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	err := row.Scan(
		&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status,
		&t.BrandingLogoURL, &t.BrandingPrimaryColor,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func scanTenantRow(rows pgx.Rows) (*domain.Tenant, error) {
	t := &domain.Tenant{}
	err := rows.Scan(
		&t.ID, &t.Slug, &t.Name, &t.Plan, &t.Status,
		&t.BrandingLogoURL, &t.BrandingPrimaryColor,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return t, nil
}
