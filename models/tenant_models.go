package models

import "time"

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
