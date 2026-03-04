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

type pgAgentConfigRepo struct{}

func NewAgentConfigRepository() AgentConfigRepository {
	return &pgAgentConfigRepo{}
}

func (r *pgAgentConfigRepo) Create(ctx context.Context, q Querier, ac *domain.AgentConfig) error {
	sql := `
		INSERT INTO agent_configs
		  (id, agent_profile_id, version, status, conversation_policy, escalation_rules,
		   tool_permissions, llm_params, channel_format_rules, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at`
	return q.QueryRow(ctx, sql,
		ac.ID, ac.AgentProfileID, ac.Version, ac.Status,
		ac.ConversationPolicy, ac.EscalationRules, ac.ToolPermissions,
		ac.LLMParams, ac.ChannelFormatRules, ac.CreatedBy,
	).Scan(&ac.CreatedAt)
}

func (r *pgAgentConfigRepo) GetByID(ctx context.Context, q Querier, id uuid.UUID) (*domain.AgentConfig, error) {
	sql := `SELECT id, agent_profile_id, version, status, conversation_policy, escalation_rules,
				   tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at
			FROM agent_configs WHERE id = $1`
	ac, err := scanAgentConfig(q.QueryRow(ctx, sql, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("agent config %s: %w", id, apperrors.ErrNotFound)
	}
	return ac, err
}

func (r *pgAgentConfigRepo) GetActive(ctx context.Context, q Querier, profileID uuid.UUID) (*domain.AgentConfig, error) {
	sql := `SELECT id, agent_profile_id, version, status, conversation_policy, escalation_rules,
				   tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at
			FROM agent_configs WHERE agent_profile_id = $1 AND status = 'active'`
	ac, err := scanAgentConfig(q.QueryRow(ctx, sql, profileID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("active config for profile %s: %w", profileID, apperrors.ErrNotFound)
	}
	return ac, err
}

func (r *pgAgentConfigRepo) ListByProfile(ctx context.Context, q Querier, profileID uuid.UUID) ([]*domain.AgentConfig, error) {
	sql := `SELECT id, agent_profile_id, version, status, conversation_policy, escalation_rules,
				   tool_permissions, llm_params, channel_format_rules, created_by, created_at, activated_at
			FROM agent_configs WHERE agent_profile_id = $1 ORDER BY version DESC`
	rows, err := q.Query(ctx, sql, profileID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []*domain.AgentConfig
	for rows.Next() {
		ac := &domain.AgentConfig{}
		if err := rows.Scan(
			&ac.ID, &ac.AgentProfileID, &ac.Version, &ac.Status,
			&ac.ConversationPolicy, &ac.EscalationRules, &ac.ToolPermissions,
			&ac.LLMParams, &ac.ChannelFormatRules, &ac.CreatedBy, &ac.CreatedAt, &ac.ActivatedAt,
		); err != nil {
			return nil, err
		}
		configs = append(configs, ac)
	}
	return configs, rows.Err()
}

func (r *pgAgentConfigRepo) Update(ctx context.Context, q Querier, ac *domain.AgentConfig) error {
	sql := `UPDATE agent_configs
			SET conversation_policy = $2, escalation_rules = $3, tool_permissions = $4,
			    llm_params = $5, channel_format_rules = $6
			WHERE id = $1 AND status = 'draft'`
	tag, err := q.Exec(ctx, sql,
		ac.ID, ac.ConversationPolicy, ac.EscalationRules,
		ac.ToolPermissions, ac.LLMParams, ac.ChannelFormatRules,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("agent config %s: %w", ac.ID, apperrors.ErrNotFound)
	}
	return nil
}

func (r *pgAgentConfigRepo) ArchiveActiveByProfile(ctx context.Context, q Querier, profileID uuid.UUID) error {
	_, err := q.Exec(ctx,
		`UPDATE agent_configs SET status = 'archived' WHERE agent_profile_id = $1 AND status = 'active'`,
		profileID,
	)
	return err
}

func (r *pgAgentConfigRepo) NextVersion(ctx context.Context, q Querier, profileID uuid.UUID) (int, error) {
	var max int
	err := q.QueryRow(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM agent_configs WHERE agent_profile_id = $1`,
		profileID,
	).Scan(&max)
	return max + 1, err
}

func scanAgentConfig(row pgx.Row) (*domain.AgentConfig, error) {
	ac := &domain.AgentConfig{}
	err := row.Scan(
		&ac.ID, &ac.AgentProfileID, &ac.Version, &ac.Status,
		&ac.ConversationPolicy, &ac.EscalationRules, &ac.ToolPermissions,
		&ac.LLMParams, &ac.ChannelFormatRules, &ac.CreatedBy, &ac.CreatedAt, &ac.ActivatedAt,
	)
	if err != nil {
		return nil, err
	}
	return ac, nil
}
