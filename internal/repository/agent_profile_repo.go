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

type pgAgentProfileRepo struct{}

func NewAgentProfileRepository() AgentProfileRepository {
	return &pgAgentProfileRepo{}
}

func (r *pgAgentProfileRepo) Create(ctx context.Context, q Querier, p *domain.AgentProfile) error {
	sql := `
		INSERT INTO agent_profiles (id, name, description, scheduling_flow_rules, escalation_rules, allowed_specialties, allowed_locations)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`
	return q.QueryRow(ctx, sql,
		p.ID, p.Name, p.Description, p.SchedulingFlowRules, p.EscalationRules,
		p.AllowedSpecialties, p.AllowedLocations,
	).Scan(&p.CreatedAt, &p.UpdatedAt)
}

func (r *pgAgentProfileRepo) GetByID(ctx context.Context, q Querier, id uuid.UUID) (*domain.AgentProfile, error) {
	sql := `SELECT id, name, description, scheduling_flow_rules, escalation_rules,
				   allowed_specialties, allowed_locations, agent_config_id, created_at, updated_at
			FROM agent_profiles WHERE id = $1`
	p, err := scanProfile(q.QueryRow(ctx, sql, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("profile %s: %w", id, apperrors.ErrNotFound)
	}
	return p, err
}

func (r *pgAgentProfileRepo) List(ctx context.Context, q Querier) ([]*domain.AgentProfile, error) {
	sql := `SELECT id, name, description, scheduling_flow_rules, escalation_rules,
				   allowed_specialties, allowed_locations, agent_config_id, created_at, updated_at
			FROM agent_profiles ORDER BY created_at DESC`
	rows, err := q.Query(ctx, sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*domain.AgentProfile
	for rows.Next() {
		p := &domain.AgentProfile{}
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Description, &p.SchedulingFlowRules, &p.EscalationRules,
			&p.AllowedSpecialties, &p.AllowedLocations, &p.AgentConfigID, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

func (r *pgAgentProfileRepo) Update(ctx context.Context, q Querier, p *domain.AgentProfile) error {
	sql := `UPDATE agent_profiles
			SET name = $2, description = $3, scheduling_flow_rules = $4,
			    escalation_rules = $5, allowed_specialties = $6, allowed_locations = $7,
			    updated_at = NOW()
			WHERE id = $1
			RETURNING updated_at`
	err := q.QueryRow(ctx, sql,
		p.ID, p.Name, p.Description, p.SchedulingFlowRules, p.EscalationRules,
		p.AllowedSpecialties, p.AllowedLocations,
	).Scan(&p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("profile %s: %w", p.ID, apperrors.ErrNotFound)
	}
	return err
}

func (r *pgAgentProfileRepo) SetActiveConfig(ctx context.Context, q Querier, profileID, configID uuid.UUID) error {
	_, err := q.Exec(ctx,
		`UPDATE agent_profiles SET agent_config_id = $2, updated_at = NOW() WHERE id = $1`,
		profileID, configID,
	)
	return err
}

func scanProfile(row pgx.Row) (*domain.AgentProfile, error) {
	p := &domain.AgentProfile{}
	err := row.Scan(
		&p.ID, &p.Name, &p.Description, &p.SchedulingFlowRules, &p.EscalationRules,
		&p.AllowedSpecialties, &p.AllowedLocations, &p.AgentConfigID, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}
