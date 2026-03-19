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

type pgToolRepo struct {
	db *pgxpool.Pool
}

func NewToolRepository(db *pgxpool.Pool) ToolRepository {
	return &pgToolRepo{db: db}
}

func (r *pgToolRepo) Create(ctx context.Context, t *domain.Tool) error {
	q := `
		INSERT INTO tool_registry (id, name, description, openai_function_def, is_active)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`
	return r.db.QueryRow(ctx, q,
		t.ID, t.Name, t.Description, t.OpenAIFunctionDef, t.IsActive,
	).Scan(&t.CreatedAt, &t.UpdatedAt)
}

func (r *pgToolRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tool, error) {
	q := `SELECT id, name, description, openai_function_def, is_active, created_at, updated_at
		  FROM tool_registry WHERE id = $1`
	t, err := scanTool(r.db.QueryRow(ctx, q, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("tool %s: %w", id, apperrors.ErrNotFound)
	}
	return t, err
}

func (r *pgToolRepo) GetByName(ctx context.Context, name string) (*domain.Tool, error) {
	q := `SELECT id, name, description, openai_function_def, is_active, created_at, updated_at
		  FROM tool_registry WHERE name = $1`
	t, err := scanTool(r.db.QueryRow(ctx, q, name))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("tool %q: %w", name, apperrors.ErrNotFound)
	}
	return t, err
}

func (r *pgToolRepo) List(ctx context.Context) ([]*domain.Tool, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description, openai_function_def, is_active, created_at, updated_at
		 FROM tool_registry ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tools []*domain.Tool
	for rows.Next() {
		t := &domain.Tool{}
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.OpenAIFunctionDef, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tools = append(tools, t)
	}
	return tools, rows.Err()
}

func (r *pgToolRepo) Update(ctx context.Context, t *domain.Tool) error {
	q := `UPDATE tool_registry
		  SET description = $2, openai_function_def = $3, is_active = $4, updated_at = NOW()
		  WHERE id = $1
		  RETURNING updated_at`
	err := r.db.QueryRow(ctx, q, t.ID, t.Description, t.OpenAIFunctionDef, t.IsActive).Scan(&t.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("tool %s: %w", t.ID, apperrors.ErrNotFound)
	}
	return err
}

func scanTool(row pgx.Row) (*domain.Tool, error) {
	t := &domain.Tool{}
	err := row.Scan(&t.ID, &t.Name, &t.Description, &t.OpenAIFunctionDef, &t.IsActive, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return t, nil
}
