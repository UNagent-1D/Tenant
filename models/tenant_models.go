package models

// CreateTenantRequest mapea los datos enviados para crear un nuevo tenant
type CreateTenantRequest struct {
	Name   string `json:"name" binding:"required"`
	Domain string `json:"domain"`
}

// Tenant engloba los datos principales de un tenant
type Tenant struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Domain   string `json:"domain"`
	IsActive bool   `json:"is_active"`
}
