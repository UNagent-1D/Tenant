package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type pgDataSourceRepo struct{}

func NewDataSourceRepository() DataSourceRepository {
	return &pgDataSourceRepo{}
}

func (r *pgDataSourceRepo) Create(ctx context.Context, q Querier, ds *domain.DataSource) error {
	sql := `
		INSERT INTO data_sources (id, name, source_type, base_url, credential_ref, route_configs, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`
	return q.QueryRow(ctx, sql,
		ds.ID, ds.Name, ds.SourceType, ds.BaseURL, ds.CredentialRef, ds.RouteConfigs, ds.IsActive,
	).Scan(&ds.CreatedAt, &ds.UpdatedAt)
}

func (r *pgDataSourceRepo) GetByID(ctx context.Context, q Querier, id uuid.UUID) (*domain.DataSource, error) {
	sql := `SELECT id, name, source_type, base_url, credential_ref, route_configs, is_active, created_at, updated_at
			FROM data_sources WHERE id = $1`
	ds, err := scanDataSource(q.QueryRow(ctx, sql, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("data source %s: %w", id, apperrors.ErrNotFound)
	}
	return ds, err
}

func (r *pgDataSourceRepo) List(ctx context.Context, q Querier) ([]*domain.DataSource, error) {
	sql := `SELECT id, name, source_type, base_url, credential_ref, route_configs, is_active, created_at, updated_at
			FROM data_sources ORDER BY created_at DESC`
	rows, err := q.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []*domain.DataSource
	for rows.Next() {
		ds := &domain.DataSource{}
		if err := rows.Scan(&ds.ID, &ds.Name, &ds.SourceType, &ds.BaseURL, &ds.CredentialRef, &ds.RouteConfigs, &ds.IsActive, &ds.CreatedAt, &ds.UpdatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, ds)
	}
	return sources, rows.Err()
}

func (r *pgDataSourceRepo) Update(ctx context.Context, q Querier, ds *domain.DataSource) error {
	sql := `UPDATE data_sources
			SET name = $2, base_url = $3, credential_ref = $4, route_configs = $5, is_active = $6, updated_at = NOW()
			WHERE id = $1
			RETURNING updated_at`
	err := q.QueryRow(ctx, sql,
		ds.ID, ds.Name, ds.BaseURL, ds.CredentialRef, ds.RouteConfigs, ds.IsActive,
	).Scan(&ds.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("data source %s: %w", ds.ID, apperrors.ErrNotFound)
	}
	return err
}

func scanDataSource(row pgx.Row) (*domain.DataSource, error) {
	ds := &domain.DataSource{}
	err := row.Scan(&ds.ID, &ds.Name, &ds.SourceType, &ds.BaseURL, &ds.CredentialRef, &ds.RouteConfigs, &ds.IsActive, &ds.CreatedAt, &ds.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return ds, nil
}
