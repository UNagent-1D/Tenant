package domain

import (
	"time"

	"github.com/google/uuid"
)

type ChannelType string

const (
	ChannelWhatsApp  ChannelType = "whatsapp"
	ChannelWebWidget ChannelType = "web_widget"
)

type Channel struct {
	ID               uuid.UUID   `json:"id"`
	TenantID         uuid.UUID   `json:"tenant_id"`
	ChannelType      ChannelType `json:"channel_type"`
	ChannelKey       string      `json:"channel_key"`
	WebhookSecretRef *string     `json:"webhook_secret_ref,omitempty"`
	IsActive         bool        `json:"is_active"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}
