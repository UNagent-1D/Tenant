CREATE TYPE channel_type AS ENUM ('whatsapp', 'web_widget');

CREATE TABLE IF NOT EXISTS channels (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID         NOT NULL,
    channel_type       channel_type NOT NULL,
    channel_key        TEXT         NOT NULL,
    webhook_secret_ref TEXT,
    is_active          BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, channel_key)
);

CREATE INDEX idx_channels_tenant_id ON channels (tenant_id);
