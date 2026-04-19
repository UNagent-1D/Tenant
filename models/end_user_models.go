package models

import "time"

type CreateEndUserRequest struct {
	FullName    string  `json:"full_name"   binding:"required"`
	NationalID  string  `json:"national_id" binding:"required"`
	Cellphone   string  `json:"cellphone"   binding:"required"`
	Email       *string `json:"email"`
	DateOfBirth *string `json:"date_of_birth"`
	ExternalRef *string `json:"external_ref"`
}

type EndUser struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id"`
	FullName    string    `json:"full_name"`
	NationalID  string    `json:"national_id"`
	Cellphone   string    `json:"cellphone"`
	Email       *string   `json:"email"`
	IsActive    bool      `json:"is_active"`
	ExternalRef *string   `json:"external_ref"`
	CreatedAt   time.Time `json:"created_at"`
}

type LookupResponse struct {
	Exists bool    `json:"exists"`
	UserID *string `json:"user_id,omitempty"`
}
