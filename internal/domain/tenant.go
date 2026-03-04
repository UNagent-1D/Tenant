package domain

import (
	"time"

	"github.com/google/uuid"
)

type TenantPlan string
type TenantStatus string

const (
	PlanFree       TenantPlan = "free"
	PlanStarter    TenantPlan = "starter"
	PlanPro        TenantPlan = "pro"
	PlanEnterprise TenantPlan = "enterprise"

	StatusActive    TenantStatus = "active"
	StatusSuspended TenantStatus = "suspended"
	StatusChurned   TenantStatus = "churned"
)

type Tenant struct {
	ID                   uuid.UUID    `json:"id"`
	Slug                 string       `json:"slug"`
	Name                 string       `json:"name"`
	Plan                 TenantPlan   `json:"plan"`
	Status               TenantStatus `json:"status"`
	BrandingLogoURL      *string      `json:"branding_logo_url,omitempty"`
	BrandingPrimaryColor *string      `json:"branding_primary_color,omitempty"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}
