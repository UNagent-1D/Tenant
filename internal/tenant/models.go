package tenant

import (
	"encoding/json"
	"time"
)

// ── Tenant ───────────────────────────────────────────────────────────────────

type Tenant struct {
	ID                   string    `json:"id"`
	Slug                 string    `json:"slug"`
	Name                 string    `json:"name"`
	Plan                 string    `json:"plan"`
	Status               string    `json:"status"`
	BrandingLogoURL      *string   `json:"branding_logo_url"`
	BrandingPrimaryColor *string   `json:"branding_primary_color"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type CreateTenantRequest struct {
	Slug                 string  `json:"slug" binding:"required"`
	Name                 string  `json:"name" binding:"required"`
	Plan                 string  `json:"plan" binding:"required,oneof=free starter pro enterprise"`
	BrandingLogoURL      *string `json:"branding_logo_url"`
	BrandingPrimaryColor *string `json:"branding_primary_color"`
}

type UpdateTenantRequest struct {
	Name                 *string `json:"name"`
	Plan                 *string `json:"plan"   binding:"omitempty,oneof=free starter pro enterprise"`
	Status               *string `json:"status" binding:"omitempty,oneof=active suspended churned"`
	BrandingLogoURL      *string `json:"branding_logo_url"`
	BrandingPrimaryColor *string `json:"branding_primary_color"`
}

// ── Channel ───────────────────────────────────────────────────────────────────

type Channel struct {
	ID               string  `json:"id"`
	TenantID         string  `json:"tenant_id"`
	ChannelType      string  `json:"channel_type"`
	ChannelKey       string  `json:"channel_key"`
	WebhookSecretRef *string `json:"webhook_secret_ref"`
	IsActive         bool    `json:"is_active"`
}

type CreateChannelRequest struct {
	ChannelType      string  `json:"channel_type"       binding:"required,oneof=whatsapp web_widget"`
	ChannelKey       string  `json:"channel_key"        binding:"required"`
	WebhookSecretRef *string `json:"webhook_secret_ref"`
}

type UpdateChannelRequest struct {
	ChannelKey       *string `json:"channel_key"`
	WebhookSecretRef *string `json:"webhook_secret_ref"`
	IsActive         *bool   `json:"is_active"`
}

// ── AgentProfile ──────────────────────────────────────────────────────────────

type AgentProfile struct {
	ID                  string          `json:"id"`
	Name                string          `json:"name"`
	Description         *string         `json:"description"`
	SchedulingFlowRules json.RawMessage `json:"scheduling_flow_rules"`
	EscalationRules     json.RawMessage `json:"escalation_rules"`
	AllowedSpecialties  []string        `json:"allowed_specialties"`
	AllowedLocations    []string        `json:"allowed_locations"`
	AgentConfigID       *string         `json:"agent_config_id"`
}

type CreateProfileRequest struct {
	Name                string          `json:"name"         binding:"required"`
	Description         *string         `json:"description"`
	SchedulingFlowRules json.RawMessage `json:"scheduling_flow_rules"`
	EscalationRules     json.RawMessage `json:"escalation_rules"`
	AllowedSpecialties  []string        `json:"allowed_specialties"`
	AllowedLocations    []string        `json:"allowed_locations"`
}

type UpdateProfileRequest struct {
	Name                *string         `json:"name"`
	Description         *string         `json:"description"`
	SchedulingFlowRules json.RawMessage `json:"scheduling_flow_rules"`
	EscalationRules     json.RawMessage `json:"escalation_rules"`
	AllowedSpecialties  []string        `json:"allowed_specialties"`
	AllowedLocations    []string        `json:"allowed_locations"`
}

// ── DataSource ────────────────────────────────────────────────────────────────

type DataSource struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	SourceType    string          `json:"source_type"`
	BaseURL       string          `json:"base_url"`
	CredentialRef *string         `json:"credential_ref"`
	RouteConfigs  json.RawMessage `json:"route_configs"`
	IsActive      bool            `json:"is_active"`
}

type CreateDataSourceRequest struct {
	Name          string          `json:"name"          binding:"required"`
	SourceType    string          `json:"source_type"   binding:"required,oneof=scheduling patient_registry"`
	BaseURL       string          `json:"base_url"      binding:"required"`
	CredentialRef *string         `json:"credential_ref"`
	RouteConfigs  json.RawMessage `json:"route_configs"  binding:"required"`
}

type UpdateDataSourceRequest struct {
	RouteConfigs  json.RawMessage `json:"route_configs"`
	CredentialRef *string         `json:"credential_ref"`
	IsActive      *bool           `json:"is_active"`
}

// ── AgentConfig ───────────────────────────────────────────────────────────────

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

// ── Tool Registry ─────────────────────────────────────────────────────────────

type Tool struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Description       *string         `json:"description"`
	OpenAIFunctionDef json.RawMessage `json:"openai_function_def"`
	IsActive          bool            `json:"is_active"`
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

// ── EndUser ───────────────────────────────────────────────────────────────────

type EndUser struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	FullName    string    `json:"full_name"`
	NationalID  string    `json:"national_id"`
	Cellphone   string    `json:"cellphone"`
	Email       *string   `json:"email"`
	IsActive    bool      `json:"is_active"`
	ExternalRef *string   `json:"external_ref"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateEndUserRequest struct {
	FullName    string  `json:"full_name"    binding:"required"`
	NationalID  string  `json:"national_id"  binding:"required"`
	Cellphone   string  `json:"cellphone"    binding:"required"`
	Email       *string `json:"email"`
	DateOfBirth *string `json:"date_of_birth"`
	ExternalRef *string `json:"external_ref"`
}

type LookupResponse struct {
	Exists bool    `json:"exists"`
	UserID *string `json:"user_id,omitempty"`
}

// ── Execute (endpoint interno) ────────────────────────────────────────────────

type ExecuteRequest struct {
	DataSourceID string            `json:"data_source_id" binding:"required"`
	Operation    string            `json:"operation"      binding:"required"`
	PathParams   map[string]string `json:"path_params"`
	QueryParams  map[string]string `json:"query_params"`
	Body         map[string]any    `json:"body"`
}

type ExecuteResponse struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       any               `json:"body"`
}
