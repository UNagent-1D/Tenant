CREATE TYPE user_role AS ENUM ('app_admin', 'tenant_admin', 'tenant_operator');

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role          user_role NOT NULL,
    tenant_id     UUID REFERENCES tenants(id) ON DELETE RESTRICT,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT app_admin_no_tenant CHECK (
        (role = 'app_admin' AND tenant_id IS NULL) OR
        (role != 'app_admin' AND tenant_id IS NOT NULL)
    )
);

CREATE INDEX idx_users_email     ON users(email);
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
