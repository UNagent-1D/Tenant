CREATE TYPE user_role AS ENUM ('app_admin', 'tenant_admin', 'tenant_operator');

CREATE TABLE IF NOT EXISTS users (
    id            UUID       PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT       NOT NULL UNIQUE,
    password_hash TEXT       NOT NULL,
    role          user_role  NOT NULL,
    tenant_id     UUID       REFERENCES tenants (id) ON DELETE SET NULL,
    is_active     BOOLEAN    NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email     ON users (email);
CREATE INDEX idx_users_tenant_id ON users (tenant_id);
