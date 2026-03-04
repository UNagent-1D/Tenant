package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type SourceType string

const (
	SourceTypeScheduling      SourceType = "scheduling"
	SourceTypePatientRegistry SourceType = "patient_registry"
)

// RouteConfig describes one logical operation on an external API.
type RouteConfig struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type DataSource struct {
	ID            uuid.UUID       `json:"id"`
	Name          string          `json:"name"`
	SourceType    SourceType      `json:"source_type"`
	BaseURL       string          `json:"base_url"`
	CredentialRef *string         `json:"credential_ref,omitempty"`
	RouteConfigs  json.RawMessage `json:"route_configs"`
	IsActive      bool            `json:"is_active"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}
