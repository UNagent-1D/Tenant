package repository

import (
	"context"

	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Querier is satisfied by both *pgxpool.Conn and pgx.Tx,
// allowing repositories to work with or without a transaction.
type Querier interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgx.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// --- Global schema repositories ---

type TenantRepository interface {
	Create(ctx context.Context, t *domain.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error)
	List(ctx context.Context, page, limit int) ([]*domain.Tenant, int, error)
	Update(ctx context.Context, t *domain.Tenant) error
	SlugExists(ctx context.Context, slug string) (bool, error)
}

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context, tenantID *uuid.UUID, page, limit int) ([]*domain.User, int, error)
	Update(ctx context.Context, u *domain.User) error
}

type ToolRepository interface {
	Create(ctx context.Context, t *domain.Tool) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Tool, error)
	GetByName(ctx context.Context, name string) (*domain.Tool, error)
	List(ctx context.Context) ([]*domain.Tool, error)
	Update(ctx context.Context, t *domain.Tool) error
}

// --- Tenant-scoped repositories (accept Querier for transaction support) ---

type ChannelRepository interface {
	Create(ctx context.Context, q Querier, c *domain.Channel) error
	GetByID(ctx context.Context, q Querier, id uuid.UUID) (*domain.Channel, error)
	List(ctx context.Context, q Querier, tenantID uuid.UUID) ([]*domain.Channel, error)
	Update(ctx context.Context, q Querier, c *domain.Channel) error
	KeyExists(ctx context.Context, q Querier, tenantID uuid.UUID, key string) (bool, error)
}

type AgentProfileRepository interface {
	Create(ctx context.Context, q Querier, p *domain.AgentProfile) error
	GetByID(ctx context.Context, q Querier, id uuid.UUID) (*domain.AgentProfile, error)
	List(ctx context.Context, q Querier) ([]*domain.AgentProfile, error)
	Update(ctx context.Context, q Querier, p *domain.AgentProfile) error
	SetActiveConfig(ctx context.Context, q Querier, profileID, configID uuid.UUID) error
}

type DataSourceRepository interface {
	Create(ctx context.Context, q Querier, ds *domain.DataSource) error
	GetByID(ctx context.Context, q Querier, id uuid.UUID) (*domain.DataSource, error)
	List(ctx context.Context, q Querier) ([]*domain.DataSource, error)
	Update(ctx context.Context, q Querier, ds *domain.DataSource) error
}

type AgentConfigRepository interface {
	Create(ctx context.Context, q Querier, ac *domain.AgentConfig) error
	GetByID(ctx context.Context, q Querier, id uuid.UUID) (*domain.AgentConfig, error)
	GetActive(ctx context.Context, q Querier, profileID uuid.UUID) (*domain.AgentConfig, error)
	ListByProfile(ctx context.Context, q Querier, profileID uuid.UUID) ([]*domain.AgentConfig, error)
	Update(ctx context.Context, q Querier, ac *domain.AgentConfig) error
	ArchiveActiveByProfile(ctx context.Context, q Querier, profileID uuid.UUID) error
	NextVersion(ctx context.Context, q Querier, profileID uuid.UUID) (int, error)
}

type EndUserRepository interface {
	Create(ctx context.Context, q Querier, eu *domain.EndUser) error
	GetByPhone(ctx context.Context, q Querier, cellphone string) (*domain.EndUser, error)
	GetByNationalID(ctx context.Context, q Querier, nationalID string) (*domain.EndUser, error)
}

// PoolQuerier wraps *pgxpool.Pool to implement Querier via a single connection acquire.
// Use AcquireTenantConn (db package) instead when a schema-scoped connection is needed.
type PoolQuerier struct {
	Pool *pgxpool.Pool
}
