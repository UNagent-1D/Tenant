package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// User holds the data scanned from the users table for authentication.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         string
	TenantID     *string
	IsActive     bool
}

type Claims struct {
	UserID   string  `json:"user_id"`
	Email    string  `json:"email"`
	TenantID *string `json:"tenant_id"`
	Role     string  `json:"role"`
	jwt.RegisteredClaims
}

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
