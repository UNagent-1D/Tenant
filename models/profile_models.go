package models

import "encoding/json"

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
