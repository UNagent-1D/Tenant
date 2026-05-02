-- Clean up old incompatible schema if it exists
DROP TABLE IF EXISTS user_tenants CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
DROP TYPE IF EXISTS system_role CASCADE;

CREATE TYPE tenant_plan AS ENUM ('free', 'starter', 'pro', 'enterprise');
CREATE TYPE tenant_status AS ENUM ('active', 'suspended', 'churned');

CREATE TABLE tenants (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug                   TEXT UNIQUE NOT NULL,
    name                   TEXT NOT NULL,
    plan                   tenant_plan NOT NULL DEFAULT 'free',
    status                 tenant_status NOT NULL DEFAULT 'active',
    branding_logo_url      TEXT,
    branding_primary_color CHAR(7),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);
