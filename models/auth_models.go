package models

import (
	"github.com/golang-jwt/jwt/v5"
)

// LoginRequest mapea los datos enviados por el usuario para autenticarse
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// User engloba los datos principales de un usuario
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Active       bool
}

// UserTenant define los atributos de rol y relación de tenant que trae la BD en el JOIN
type UserTenant struct {
	TenantID *string // Es un puntero porque si el rol es app_admin, el tenant_id es NULL
	Role     string
}

// Claims representa el payload que formará el JWT con los roles y el tenant
type Claims struct {
	UserID   string  `json:"user_id"`
	Email    string  `json:"email"`
	TenantID *string `json:"tenant_id"`
	Role     string  `json:"role"`
	jwt.RegisteredClaims
}
