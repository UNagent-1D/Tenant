package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/UNagent-1D/tenant-service/internal/apperrors"
	"github.com/UNagent-1D/tenant-service/internal/db"
	"github.com/UNagent-1D/tenant-service/internal/domain"
	"github.com/UNagent-1D/tenant-service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AgentConfigService interface {
	Create(ctx context.Context, slug string, profileID uuid.UUID, createdBy uuid.UUID, req CreateAgentConfigRequest) (*domain.AgentConfig, error)
	List(ctx context.Context, slug string, profileID uuid.UUID) ([]*domain.AgentConfig, error)
	GetActive(ctx context.Context, slug string, profileID uuid.UUID) (*domain.AgentConfig, error)
	Update(ctx context.Context, slug string, configID uuid.UUID, req UpdateAgentConfigRequest) (*domain.AgentConfig, error)
	Activate(ctx context.Context, slug string, profileID, configID uuid.UUID) (*domain.AgentConfig, error)
}

type llmParams struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
	SystemPrompt string `json:"system_prompt"`
}

type toolPermission struct {
	ToolName    string          `json:"tool_name"`
	Constraints json.RawMessage `json:"constraints"`
}

type escalationRulesCheck struct {
	Triggers []interface{} `json:"triggers"`
}

type CreateAgentConfigRequest struct {
	ConversationPolicy  json.RawMessage `json:"conversation_policy" binding:"required"`
	EscalationRules     json.RawMessage `json:"escalation_rules" binding:"required"`
	ToolPermissions     json.RawMessage `json:"tool_permissions" binding:"required"`
	LLMParams           json.RawMessage `json:"llm_params" binding:"required"`
	ChannelFormatRules  json.RawMessage `json:"channel_format_rules"`
}

type UpdateAgentConfigRequest struct {
	ConversationPolicy  json.RawMessage `json:"conversation_policy"`
	EscalationRules     json.RawMessage `json:"escalation_rules"`
	ToolPermissions     json.RawMessage `json:"tool_permissions"`
	LLMParams           json.RawMessage `json:"llm_params"`
	ChannelFormatRules  json.RawMessage `json:"channel_format_rules"`
}

type agentConfigService struct {
	repo     repository.AgentConfigRepository
	profRepo repository.AgentProfileRepository
	toolRepo repository.ToolRepository
	pool     *pgxpool.Pool
}

func NewAgentConfigService(
	repo repository.AgentConfigRepository,
	profRepo repository.AgentProfileRepository,
	toolRepo repository.ToolRepository,
	pool *pgxpool.Pool,
) AgentConfigService {
	return &agentConfigService{repo: repo, profRepo: profRepo, toolRepo: toolRepo, pool: pool}
}

func (s *agentConfigService) Create(ctx context.Context, slug string, profileID uuid.UUID, createdBy uuid.UUID, req CreateAgentConfigRequest) (*domain.AgentConfig, error) {
	if err := s.validateConfig(ctx, req.LLMParams, req.ToolPermissions, req.EscalationRules); err != nil {
		return nil, err
	}

	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Ensure the profile exists
	if _, err := s.profRepo.GetByID(ctx, conn, profileID); err != nil {
		return nil, err
	}

	version, err := s.repo.NextVersion(ctx, conn, profileID)
	if err != nil {
		return nil, err
	}

	ac := &domain.AgentConfig{
		ID:                 uuid.New(),
		AgentProfileID:     profileID,
		Version:            version,
		Status:             domain.ConfigDraft,
		ConversationPolicy: req.ConversationPolicy,
		EscalationRules:    req.EscalationRules,
		ToolPermissions:    req.ToolPermissions,
		LLMParams:          req.LLMParams,
		ChannelFormatRules: req.ChannelFormatRules,
		CreatedBy:          &createdBy,
	}

	if err := s.repo.Create(ctx, conn, ac); err != nil {
		return nil, err
	}
	return ac, nil
}

func (s *agentConfigService) List(ctx context.Context, slug string, profileID uuid.UUID) ([]*domain.AgentConfig, error) {
	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	return s.repo.ListByProfile(ctx, conn, profileID)
}

func (s *agentConfigService) GetActive(ctx context.Context, slug string, profileID uuid.UUID) (*domain.AgentConfig, error) {
	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	return s.repo.GetActive(ctx, conn, profileID)
}

func (s *agentConfigService) Update(ctx context.Context, slug string, configID uuid.UUID, req UpdateAgentConfigRequest) (*domain.AgentConfig, error) {
	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	ac, err := s.repo.GetByID(ctx, conn, configID)
	if err != nil {
		return nil, err
	}
	if ac.Status != domain.ConfigDraft {
		return nil, fmt.Errorf("only draft configs can be updated: %w", apperrors.ErrValidation)
	}

	if req.ConversationPolicy != nil {
		ac.ConversationPolicy = req.ConversationPolicy
	}
	if req.EscalationRules != nil {
		ac.EscalationRules = req.EscalationRules
	}
	if req.ToolPermissions != nil {
		ac.ToolPermissions = req.ToolPermissions
	}
	if req.LLMParams != nil {
		ac.LLMParams = req.LLMParams
	}
	if req.ChannelFormatRules != nil {
		ac.ChannelFormatRules = req.ChannelFormatRules
	}

	if err := s.validateConfig(ctx, ac.LLMParams, ac.ToolPermissions, ac.EscalationRules); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, conn, ac); err != nil {
		return nil, err
	}
	return ac, nil
}

// Activate atomically archives the current active config and promotes the target draft.
func (s *agentConfigService) Activate(ctx context.Context, slug string, profileID, configID uuid.UUID) (*domain.AgentConfig, error) {
	conn, err := db.AcquireTenantConn(ctx, s.pool, slug)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Get the target config and verify it is a draft
	target, err := s.repo.GetByID(ctx, conn, configID)
	if err != nil {
		return nil, err
	}
	if target.Status != domain.ConfigDraft {
		return nil, fmt.Errorf("only a draft config can be activated: %w", apperrors.ErrValidation)
	}
	if target.AgentProfileID != profileID {
		return nil, fmt.Errorf("config not found for this profile: %w", apperrors.ErrNotFound)
	}

	// Atomic: archive current active + activate target
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	prevActive, _ := s.repo.GetActive(ctx, tx, profileID) // may be nil on first activation

	if err := s.repo.ArchiveActiveByProfile(ctx, tx, profileID); err != nil {
		return nil, err
	}

	target.Status = domain.ConfigActive
	if err := s.repo.Update(ctx, tx, target); err != nil {
		return nil, err
	}

	// Point the profile's agent_config_id to the new active config
	if err := s.profRepo.SetActiveConfig(ctx, tx, profileID, configID); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// Re-fetch to get the activated_at timestamp
	target, err = s.repo.GetByID(ctx, conn, configID)
	if err != nil {
		return nil, err
	}

	if prevActive != nil {
		target.ChannelFormatRules = []byte(fmt.Sprintf(`{"previously_active_version":%d}`, prevActive.Version))
	}
	return target, nil
}

// validateConfig validates LLM params, tool permissions, and escalation rules.
func (s *agentConfigService) validateConfig(ctx context.Context, llmRaw, toolsRaw, escalationRaw json.RawMessage) error {
	// Validate LLM params
	var params llmParams
	if err := json.Unmarshal(llmRaw, &params); err != nil {
		return fmt.Errorf("llm_params must be valid JSON: %w", apperrors.ErrValidation)
	}
	if !domain.AllowedModels[params.Model] {
		models := "gpt-4o, gpt-4o-mini"
		return fmt.Errorf("model %q is not in the allowed model list: [%s]: %w", params.Model, models, apperrors.ErrValidation)
	}
	if params.Temperature < 0.0 || params.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0.0 and 2.0, got: %.2f: %w", params.Temperature, apperrors.ErrValidation)
	}
	if params.MaxTokens < 1 || params.MaxTokens > 4096 {
		return fmt.Errorf("max_tokens must be between 1 and 4096: %w", apperrors.ErrValidation)
	}

	// Validate escalation rules have at least one trigger
	var esc escalationRulesCheck
	if err := json.Unmarshal(escalationRaw, &esc); err != nil || len(esc.Triggers) == 0 {
		return fmt.Errorf("escalation_rules must have at least one trigger: %w", apperrors.ErrValidation)
	}

	// Validate tool permissions
	var perms []toolPermission
	if err := json.Unmarshal(toolsRaw, &perms); err != nil {
		return fmt.Errorf("tool_permissions must be a JSON array: %w", apperrors.ErrValidation)
	}
	for _, p := range perms {
		tool, err := s.toolRepo.GetByName(ctx, p.ToolName)
		if err != nil {
			return fmt.Errorf("tool_name %q is not registered in the tool registry: %w", p.ToolName, apperrors.ErrValidation)
		}
		if !tool.IsActive {
			return fmt.Errorf("tool %q is currently deactivated globally: %w", p.ToolName, apperrors.ErrValidation)
		}
	}
	return nil
}
