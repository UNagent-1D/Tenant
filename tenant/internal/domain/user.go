package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleAppAdmin        UserRole = "app_admin"
	RoleTenantAdmin     UserRole = "tenant_admin"
	RoleTenantOperator  UserRole = "tenant_operator"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         UserRole   `json:"role"`
	TenantID     *uuid.UUID `json:"tenant_id,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
