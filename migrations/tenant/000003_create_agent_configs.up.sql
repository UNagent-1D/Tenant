CREATE TYPE config_status AS ENUM ('draft', 'active', 'archived');

CREATE TABLE agent_configs (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_profile_id     UUID NOT NULL REFERENCES agent_profiles(id) ON DELETE RESTRICT,
    version              INTEGER NOT NULL,
    status               config_status NOT NULL DEFAULT 'draft',
    conversation_policy  JSONB NOT NULL,
    escalation_rules     JSONB NOT NULL,
    tool_permissions     JSONB NOT NULL,
    llm_params           JSONB NOT NULL,
    channel_format_rules JSONB,
    created_by           UUID,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    activated_at         TIMESTAMPTZ,

    UNIQUE(agent_profile_id, version)
);

CREATE INDEX idx_agent_configs_profile_status ON agent_configs(agent_profile_id, status);
