package models

type CreateChannelRequest struct {
	ChannelType      string  `json:"channel_type"       binding:"required,oneof=whatsapp web_widget"`
	ChannelKey       string  `json:"channel_key"        binding:"required"`
	WebhookSecretRef *string `json:"webhook_secret_ref"`
}

type UpdateChannelRequest struct {
	ChannelKey       *string `json:"channel_key"`
	WebhookSecretRef *string `json:"webhook_secret_ref"`
	IsActive         *bool   `json:"is_active"`
}

type Channel struct {
	ID               string  `json:"id"`
	TenantID         string  `json:"tenant_id"`
	ChannelType      string  `json:"channel_type"`
	ChannelKey       string  `json:"channel_key"`
	WebhookSecretRef *string `json:"webhook_secret_ref"`
	IsActive         bool    `json:"is_active"`
}
