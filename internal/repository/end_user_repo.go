package repository

import (
	"context"
	"errors"

	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/jackc/pgx/v5"
)

type pgEndUserRepo struct{}

func NewEndUserRepository() EndUserRepository {
	return &pgEndUserRepo{}
}

func (r *pgEndUserRepo) Create(ctx context.Context, q Querier, eu *domain.EndUser) error {
	sql := `
		INSERT INTO end_users (id, full_name, national_id, cellphone, email, date_of_birth, external_ref, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at`
	return q.QueryRow(ctx, sql,
		eu.ID, eu.FullName, eu.NationalID, eu.Cellphone,
		eu.Email, eu.DateOfBirth, eu.ExternalRef, eu.IsActive,
	).Scan(&eu.CreatedAt)
}

func (r *pgEndUserRepo) GetByPhone(ctx context.Context, q Querier, cellphone string) (*domain.EndUser, error) {
	sql := `SELECT id, full_name, national_id, cellphone, email, date_of_birth, external_ref, is_active, created_at
			FROM end_users WHERE cellphone = $1`
	eu, err := scanEndUser(q.QueryRow(ctx, sql, cellphone))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // caller interprets nil as "not found" (anti-enumeration: no ErrNotFound)
	}
	return eu, err
}

func (r *pgEndUserRepo) GetByNationalID(ctx context.Context, q Querier, nationalID string) (*domain.EndUser, error) {
	sql := `SELECT id, full_name, national_id, cellphone, email, date_of_birth, external_ref, is_active, created_at
			FROM end_users WHERE national_id = $1`
	eu, err := scanEndUser(q.QueryRow(ctx, sql, nationalID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil // anti-enumeration: always return 200
	}
	return eu, err
}

func scanEndUser(row pgx.Row) (*domain.EndUser, error) {
	eu := &domain.EndUser{}
	err := row.Scan(
		&eu.ID, &eu.FullName, &eu.NationalID, &eu.Cellphone,
		&eu.Email, &eu.DateOfBirth, &eu.ExternalRef, &eu.IsActive, &eu.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return eu, nil
}
