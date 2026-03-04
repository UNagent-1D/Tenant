CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE tenant_plan AS ENUM ('free', 'starter', 'pro', 'enterprise');
CREATE TYPE tenant_status AS ENUM ('active', 'suspended', 'churned');

CREATE TABLE IF NOT EXISTS tenants (
    id                    UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                  TEXT           NOT NULL UNIQUE,
    name                  TEXT           NOT NULL,
    plan                  tenant_plan    NOT NULL DEFAULT 'free',
    status                tenant_status  NOT NULL DEFAULT 'active',
    branding_logo_url     TEXT,
    branding_primary_color CHAR(7),
    created_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tenants_slug   ON tenants (slug);
CREATE INDEX idx_tenants_status ON tenants (status);
