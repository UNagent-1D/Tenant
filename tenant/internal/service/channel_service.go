package service

import (
	"context"
	"fmt"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/db"
	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/UNagent-1D/tenant-service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ChannelService interface {
	Create(ctx context.Context, tenantID uuid.UUID, slug string, req CreateChannelRequest) (*domain.Channel, error)
	List(ctx context.Context, tenantID uuid.UUID, slug string) ([]*domain.Channel, error)
	Update(ctx context.Context, tenantID uuid.UUID, slug string, channelID uuid.UUID, req UpdateChannelRequest) (*domain.Channel, error)
}

type CreateChannelRequest struct {
	ChannelType      domain.ChannelType `json:"channel_type" binding:"required"`
	ChannelKey       string             `json:"channel_key" binding:"required"`
	WebhookSecretRef *string            `json:"webhook_secret_ref"`
}

type UpdateChannelRequest struct {
	ChannelKey       *string `json:"channel_key"`
	WebhookSecretRef *string `json:"webhook_secret_ref"`
	IsActive         *bool   `json:"is_active"`
}

type channelService struct {
	repo repository.ChannelRepository
	pool *pgxpool.Pool
}

func NewChannelService(repo repository.ChannelRepository, pool *pgxpool.Pool) ChannelService {
	return &channelService{repo: repo, pool: pool}
}

func (s *channelService) Create(ctx context.Context, tenantID uuid.UUID, slug string, req CreateChannelRequest) (*domain.Channel, error) {
	if req.ChannelType != domain.ChannelWhatsApp && req.ChannelType != domain.ChannelWebWidget {
		return nil, fmt.Errorf("channel_type must be one of: whatsapp, web_widget: %w", apperrors.ErrValidation)
	}

	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	exists, err := s.repo.KeyExists(ctx, conn, tenantID, req.ChannelKey)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("channel_key %s is already registered for this tenant: %w", req.ChannelKey, apperrors.ErrConflict)
	}

	ch := &domain.Channel{
		ID:               uuid.New(),
		TenantID:         tenantID,
		ChannelType:      req.ChannelType,
		ChannelKey:       req.ChannelKey,
		WebhookSecretRef: req.WebhookSecretRef,
		IsActive:         true,
	}
	if err := s.repo.Create(ctx, conn, ch); err != nil {
		return nil, err
	}
	return ch, nil
}

func (s *channelService) List(ctx context.Context, tenantID uuid.UUID, slug string) ([]*domain.Channel, error) {
	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	return s.repo.List(ctx, conn, tenantID)
}

func (s *channelService) Update(ctx context.Context, tenantID uuid.UUID, slug string, channelID uuid.UUID, req UpdateChannelRequest) (*domain.Channel, error) {
	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ch, err := s.repo.GetByID(ctx, conn, channelID)
	if err != nil {
		return nil, err
	}
	if ch.TenantID != tenantID {
		return nil, fmt.Errorf("channel not found for this tenant: %w", apperrors.ErrNotFound)
	}

	if req.ChannelKey != nil {
		ch.ChannelKey = *req.ChannelKey
	}
	if req.WebhookSecretRef != nil {
		ch.WebhookSecretRef = req.WebhookSecretRef
	}
	if req.IsActive != nil {
		ch.IsActive = *req.IsActive
	}

	if err := s.repo.Update(ctx, conn, ch); err != nil {
		return nil, err
	}
	return ch, nil
}
