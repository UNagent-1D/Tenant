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

type pgUserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &pgUserRepo{db: db}
}

func (r *pgUserRepo) Create(ctx context.Context, u *domain.User) error {
	q := `
		INSERT INTO users (id, email, password_hash, role, tenant_id, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, q,
		u.ID, u.Email, u.PasswordHash, u.Role, u.TenantID, u.IsActive,
	).Scan(&u.CreatedAt, &u.UpdatedAt)
}

func (r *pgUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	q := `SELECT id, email, password_hash, role, tenant_id, is_active, created_at, updated_at
		  FROM users WHERE id = $1`
	u, err := scanUser(r.db.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("user %s: %w", id, apperrors.ErrNotFound)
	}
	return u, err
}

func (r *pgUserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	q := `SELECT id, email, password_hash, role, tenant_id, is_active, created_at, updated_at
		  FROM users WHERE email = $1`
	u, err := scanUser(r.db.QueryRow(ctx, q, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("user %q: %w", email, apperrors.ErrNotFound)
	}
	return u, err
}

func (r *pgUserRepo) List(ctx context.Context, tenantID *uuid.UUID, page, limit int) ([]*domain.User, int, error) {
	offset := (page - 1) * limit

	var total int
	var countErr error
	if tenantID != nil {
		countErr = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE tenant_id = $1`, *tenantID).Scan(&total)
	} else {
		countErr = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&total)
	}
	if countErr != nil {
		return nil, 0, countErr
	}

	var rows pgx.Rows
	var err error
	if tenantID != nil {
		rows, err = r.db.Query(ctx,
			`SELECT id, email, password_hash, role, tenant_id, is_active, created_at, updated_at
			 FROM users WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			*tenantID, limit, offset)
	} else {
		rows, err = r.db.Query(ctx,
			`SELECT id, email, password_hash, role, tenant_id, is_active, created_at, updated_at
			 FROM users ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
			limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TenantID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

func (r *pgUserRepo) Update(ctx context.Context, u *domain.User) error {
	q := `UPDATE users
		  SET role = $2, is_active = $3, updated_at = NOW()
		  WHERE id = $1
		  RETURNING updated_at`
	err := r.db.QueryRow(ctx, q, u.ID, u.Role, u.IsActive).Scan(&u.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("user %s: %w", u.ID, apperrors.ErrNotFound)
	}
	return err
}

func scanUser(row pgx.Row) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.TenantID, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}
