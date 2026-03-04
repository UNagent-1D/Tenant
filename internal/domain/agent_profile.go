package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type AgentProfile struct {
	ID                   uuid.UUID       `json:"id"`
	Name                 string          `json:"name"`
	Description          *string         `json:"description,omitempty"`
	SchedulingFlowRules  json.RawMessage `json:"scheduling_flow_rules"`
	EscalationRules      json.RawMessage `json:"escalation_rules"`
	AllowedSpecialties   []string        `json:"allowed_specialties"`
	AllowedLocations     []string        `json:"allowed_locations"`
	AgentConfigID        *uuid.UUID      `json:"agent_config_id,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
}
