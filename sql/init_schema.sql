-- ============================================================
-- GLOBAL ENUMs
-- ============================================================
CREATE TYPE tenant_plan   AS ENUM ('free', 'starter', 'pro', 'enterprise');
CREATE TYPE tenant_status AS ENUM ('active', 'suspended', 'churned');
CREATE TYPE system_role   AS ENUM ('app_admin', 'tenant_admin', 'tenant_operator');

-- ============================================================
-- GLOBAL TABLES
-- ============================================================

CREATE TABLE tenants (
    id                    UUID    DEFAULT gen_random_uuid() PRIMARY KEY,
    slug                  TEXT    UNIQUE NOT NULL,
    name                  TEXT    NOT NULL,
    plan                  tenant_plan   NOT NULL DEFAULT 'free',
    status                tenant_status NOT NULL DEFAULT 'active',
    branding_logo_url     TEXT,
    branding_primary_color CHAR(7),
    created_at            TIMESTAMPTZ DEFAULT NOW(),
    updated_at            TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE users (
    id            UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    email         TEXT        UNIQUE NOT NULL,
    password_hash TEXT        NOT NULL,
    role          system_role NOT NULL,
    tenant_id     UUID        REFERENCES tenants(id) ON DELETE CASCADE,
    is_active     BOOLEAN     DEFAULT true,
    created_at    TIMESTAMPTZ DEFAULT NOW(),
    updated_at    TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT check_role_tenant CHECK (
        (role = 'app_admin'  AND tenant_id IS NULL) OR
        (role IN ('tenant_admin', 'tenant_operator') AND tenant_id IS NOT NULL)
    )
);

CREATE TABLE tool_registry (
    id                  UUID    DEFAULT gen_random_uuid() PRIMARY KEY,
    name                TEXT    UNIQUE NOT NULL,
    description         TEXT,
    openai_function_def JSONB   NOT NULL,
    is_active           BOOLEAN DEFAULT true
);

-- ============================================================
-- PER-TENANT SCHEMA PROVISIONING
-- Called once when a tenant is created.
-- Creates schema tenant_<slug> with all per-tenant tables.
-- ============================================================
CREATE OR REPLACE FUNCTION provision_tenant_schema(p_slug TEXT)
RETURNS VOID AS $$
DECLARE
    s TEXT := 'tenant_' || p_slug;
BEGIN
    EXECUTE format('CREATE SCHEMA IF NOT EXISTS %I', s);

    -- channels
    EXECUTE format($t$
        CREATE TABLE %I.channels (
            id                UUID    DEFAULT gen_random_uuid() PRIMARY KEY,
            tenant_id         UUID    NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
            channel_type      TEXT    NOT NULL CHECK (channel_type IN ('whatsapp', 'web_widget')),
            channel_key       TEXT    NOT NULL,
            webhook_secret_ref TEXT,
            is_active         BOOLEAN DEFAULT true
        )
    $t$, s);

    -- agent_profiles
    EXECUTE format($t$
        CREATE TABLE %I.agent_profiles (
            id                    UUID    DEFAULT gen_random_uuid() PRIMARY KEY,
            name                  TEXT    NOT NULL,
            description           TEXT,
            scheduling_flow_rules JSONB,
            escalation_rules      JSONB,
            allowed_specialties   TEXT[],
            allowed_locations     TEXT[],
            agent_config_id       UUID
        )
    $t$, s);

    -- agent_configs
    EXECUTE format($t$
        CREATE TABLE %I.agent_configs (
            id                  UUID    DEFAULT gen_random_uuid() PRIMARY KEY,
            agent_profile_id    UUID    NOT NULL,
            version             INTEGER NOT NULL,
            status              TEXT    NOT NULL DEFAULT 'draft'
                                CHECK (status IN ('draft', 'active', 'archived')),
            conversation_policy JSONB   NOT NULL,
            escalation_rules    JSONB   NOT NULL,
            tool_permissions    JSONB   NOT NULL,
            llm_params          JSONB   NOT NULL,
            channel_format_rules JSONB,
            created_by          UUID    REFERENCES public.users(id),
            created_at          TIMESTAMPTZ DEFAULT NOW(),
            activated_at        TIMESTAMPTZ,
            UNIQUE (agent_profile_id, version)
        )
    $t$, s);

    -- only one active config per profile
    EXECUTE format(
        'CREATE UNIQUE INDEX ON %I.agent_configs(agent_profile_id) WHERE status = ''active''',
        s
    );

    -- data_sources
    EXECUTE format($t$
        CREATE TABLE %I.data_sources (
            id             UUID    DEFAULT gen_random_uuid() PRIMARY KEY,
            name           TEXT    NOT NULL,
            source_type    TEXT    NOT NULL CHECK (source_type IN ('scheduling', 'patient_registry')),
            base_url       TEXT    NOT NULL,
            credential_ref TEXT,
            route_configs  JSONB   NOT NULL,
            is_active      BOOLEAN DEFAULT true
        )
    $t$, s);

    -- end_users
    EXECUTE format($t$
        CREATE TABLE %I.end_users (
            id           UUID    DEFAULT gen_random_uuid() PRIMARY KEY,
            tenant_id    UUID    NOT NULL REFERENCES public.tenants(id),
            full_name    TEXT    NOT NULL,
            national_id  TEXT    NOT NULL,
            cellphone    TEXT    NOT NULL,
            email        TEXT,
            date_of_birth DATE,
            is_active    BOOLEAN DEFAULT true,
            external_ref TEXT,
            created_at   TIMESTAMPTZ DEFAULT NOW(),
            updated_at   TIMESTAMPTZ DEFAULT NOW(),
            UNIQUE (tenant_id, national_id),
            UNIQUE (tenant_id, cellphone)
        )
    $t$, s);
END;
$$ LANGUAGE plpgsql;
