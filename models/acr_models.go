package models

import (
	"encoding/json"
	"time"
)

type CreateAgentConfigRequest struct {
	ConversationPolicy json.RawMessage `json:"conversation_policy" binding:"required"`
	EscalationRules    json.RawMessage `json:"escalation_rules"    binding:"required"`
	ToolPermissions    json.RawMessage `json:"tool_permissions"    binding:"required"`
	LLMParams          json.RawMessage `json:"llm_params"          binding:"required"`
	ChannelFormatRules json.RawMessage `json:"channel_format_rules"`
}

type UpdateAgentConfigRequest struct {
	ConversationPolicy json.RawMessage `json:"conversation_policy"`
	EscalationRules    json.RawMessage `json:"escalation_rules"`
	ToolPermissions    json.RawMessage `json:"tool_permissions"`
	LLMParams          json.RawMessage `json:"llm_params"`
	ChannelFormatRules json.RawMessage `json:"channel_format_rules"`
}

type AgentConfig struct {
	ID                 string          `json:"id"`
	AgentProfileID     string          `json:"agent_profile_id"`
	Version            int             `json:"version"`
	Status             string          `json:"status"`
	ConversationPolicy json.RawMessage `json:"conversation_policy"`
	EscalationRules    json.RawMessage `json:"escalation_rules"`
	ToolPermissions    json.RawMessage `json:"tool_permissions"`
	LLMParams          json.RawMessage `json:"llm_params"`
	ChannelFormatRules json.RawMessage `json:"channel_format_rules"`
	CreatedBy          *string         `json:"created_by"`
	CreatedAt          time.Time       `json:"created_at"`
	ActivatedAt        *time.Time      `json:"activated_at"`
}

type CreateToolRequest struct {
	Name              string          `json:"name"                binding:"required"`
	Description       *string         `json:"description"`
	OpenAIFunctionDef json.RawMessage `json:"openai_function_def" binding:"required"`
}

type UpdateToolRequest struct {
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

type Tool struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Description       *string         `json:"description"`
	OpenAIFunctionDef json.RawMessage `json:"openai_function_def"`
	IsActive          bool            `json:"is_active"`
}
