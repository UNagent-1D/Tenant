package models

import "time"

type CreateUserRequest struct {
	Email    string  `json:"email"    binding:"required,email"`
	Password string  `json:"password" binding:"required,min=6"`
	Role     string  `json:"role"     binding:"required,oneof=app_admin tenant_admin tenant_operator"`
	TenantID *string `json:"tenant_id"`
}

type UpdateUserRequest struct {
	Role     *string `json:"role"      binding:"omitempty,oneof=app_admin tenant_admin tenant_operator"`
	IsActive *bool   `json:"is_active"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	TenantID  *string   `json:"tenant_id"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}
