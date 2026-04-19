package models

import "github.com/golang-jwt/jwt/v5"

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
