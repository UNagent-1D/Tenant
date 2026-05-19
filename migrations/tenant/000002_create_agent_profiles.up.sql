CREATE TABLE agent_profiles (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  TEXT NOT NULL,
    description           TEXT,
    scheduling_flow_rules JSONB,
    escalation_rules      JSONB,
    allowed_specialties   TEXT[] NOT NULL DEFAULT '{}',
    allowed_locations     TEXT[] NOT NULL DEFAULT '{}',
    agent_config_id       UUID,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);
