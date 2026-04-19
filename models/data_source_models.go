package models

import "encoding/json"

type CreateDataSourceRequest struct {
	Name          string          `json:"name"         binding:"required"`
	SourceType    string          `json:"source_type"  binding:"required,oneof=scheduling patient_registry"`
	BaseURL       string          `json:"base_url"     binding:"required"`
	CredentialRef *string         `json:"credential_ref"`
	RouteConfigs  json.RawMessage `json:"route_configs" binding:"required"`
}

type UpdateDataSourceRequest struct {
	RouteConfigs  json.RawMessage `json:"route_configs"`
	CredentialRef *string         `json:"credential_ref"`
	IsActive      *bool           `json:"is_active"`
}

type DataSource struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	SourceType    string          `json:"source_type"`
	BaseURL       string          `json:"base_url"`
	CredentialRef *string         `json:"credential_ref"`
	RouteConfigs  json.RawMessage `json:"route_configs"`
	IsActive      bool            `json:"is_active"`
}
