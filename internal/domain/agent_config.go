package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ConfigStatus string

const (
	ConfigDraft    ConfigStatus = "draft"
	ConfigActive   ConfigStatus = "active"
	ConfigArchived ConfigStatus = "archived"
)

// AllowedModels is the MVP allowlist for LLM model names.
var AllowedModels = map[string]bool{
	"gpt-4o":      true,
	"gpt-4o-mini": true,
}

type AgentConfig struct {
	ID                  uuid.UUID       `json:"id"`
	AgentProfileID      uuid.UUID       `json:"agent_profile_id"`
	Version             int             `json:"version"`
	Status              ConfigStatus    `json:"status"`
	ConversationPolicy  json.RawMessage `json:"conversation_policy"`
	EscalationRules     json.RawMessage `json:"escalation_rules"`
	ToolPermissions     json.RawMessage `json:"tool_permissions"`
	LLMParams           json.RawMessage `json:"llm_params"`
	ChannelFormatRules  json.RawMessage `json:"channel_format_rules,omitempty"`
	CreatedBy           *uuid.UUID      `json:"created_by,omitempty"`
	CreatedAt           time.Time       `json:"created_at"`
	ActivatedAt         *time.Time      `json:"activated_at,omitempty"`
}
