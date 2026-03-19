CREATE TABLE IF NOT EXISTS agent_profiles (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  TEXT        NOT NULL,
    description           TEXT,
    scheduling_flow_rules JSONB       NOT NULL DEFAULT '{}',
    escalation_rules      JSONB       NOT NULL DEFAULT '{}',
    allowed_specialties   TEXT[]      NOT NULL DEFAULT '{}',
    allowed_locations     TEXT[]      NOT NULL DEFAULT '{}',
    agent_config_id       UUID,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
