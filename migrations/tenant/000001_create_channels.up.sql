CREATE TYPE channel_type AS ENUM ('whatsapp', 'web_widget');

CREATE TABLE channels (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    channel_type       channel_type NOT NULL,
    channel_key        TEXT NOT NULL,
    webhook_secret_ref TEXT,
    is_active          BOOLEAN NOT NULL DEFAULT true,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(tenant_id, channel_key)
);
