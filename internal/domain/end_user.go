package domain

import (
	"time"

	"github.com/google/uuid"
)

type EndUser struct {
	ID          uuid.UUID  `json:"id"`
	FullName    string     `json:"full_name"`
	NationalID  *string    `json:"national_id,omitempty"`
	Cellphone   *string    `json:"cellphone,omitempty"`
	Email       *string    `json:"email,omitempty"`
	DateOfBirth *string    `json:"date_of_birth,omitempty"`
	ExternalRef *string    `json:"external_ref,omitempty"`
	IsActive    bool       `json:"is_active"`
	CreatedAt   time.Time  `json:"created_at"`
}
