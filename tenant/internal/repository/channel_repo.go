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

type pgChannelRepo struct{}

func NewChannelRepository() ChannelRepository {
	return &pgChannelRepo{}
}

func (r *pgChannelRepo) Create(ctx context.Context, q Querier, c *domain.Channel) error {
	sql := `
		INSERT INTO channels (id, tenant_id, channel_type, channel_key, webhook_secret_ref, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`
	return q.QueryRow(ctx, sql,
		c.ID, c.TenantID, c.ChannelType, c.ChannelKey, c.WebhookSecretRef, c.IsActive,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

func (r *pgChannelRepo) GetByID(ctx context.Context, q Querier, id uuid.UUID) (*domain.Channel, error) {
	sql := `SELECT id, tenant_id, channel_type, channel_key, webhook_secret_ref, is_active, created_at, updated_at
			FROM channels WHERE id = $1`
	c, err := scanChannel(q.QueryRow(ctx, sql, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("channel %s: %w", id, apperrors.ErrNotFound)
	}
	return c, err
}

func (r *pgChannelRepo) List(ctx context.Context, q Querier, tenantID uuid.UUID) ([]*domain.Channel, error) {
	sql := `SELECT id, tenant_id, channel_type, channel_key, webhook_secret_ref, is_active, created_at, updated_at
			FROM channels WHERE tenant_id = $1 ORDER BY created_at DESC`
	rows, err := q.Query(ctx, sql, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []*domain.Channel
	for rows.Next() {
		c := &domain.Channel{}
		if err := rows.Scan(&c.ID, &c.TenantID, &c.ChannelType, &c.ChannelKey, &c.WebhookSecretRef, &c.IsActive, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, c)
	}
	return channels, rows.Err()
}

func (r *pgChannelRepo) Update(ctx context.Context, q Querier, c *domain.Channel) error {
	sql := `UPDATE channels
			SET channel_key = $2, webhook_secret_ref = $3, is_active = $4, updated_at = NOW()
			WHERE id = $1
			RETURNING updated_at`
	err := q.QueryRow(ctx, sql, c.ID, c.ChannelKey, c.WebhookSecretRef, c.IsActive).Scan(&c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("channel %s: %w", c.ID, apperrors.ErrNotFound)
	}
	return err
}

func (r *pgChannelRepo) KeyExists(ctx context.Context, q Querier, tenantID uuid.UUID, key string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM channels WHERE tenant_id = $1 AND channel_key = $2)`,
		tenantID, key,
	).Scan(&exists)
	return exists, err
}

func scanChannel(row pgx.Row) (*domain.Channel, error) {
	c := &domain.Channel{}
	err := row.Scan(&c.ID, &c.TenantID, &c.ChannelType, &c.ChannelKey, &c.WebhookSecretRef, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}
