CREATE TYPE config_status AS ENUM ('draft', 'active', 'archived');

CREATE TABLE IF NOT EXISTS agent_configs (
    id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_profile_id     UUID          NOT NULL REFERENCES agent_profiles (id) ON DELETE CASCADE,
    version              INTEGER       NOT NULL DEFAULT 1,
    status               config_status NOT NULL DEFAULT 'draft',
    conversation_policy  JSONB         NOT NULL DEFAULT '{}',
    escalation_rules     JSONB         NOT NULL DEFAULT '{}',
    tool_permissions     JSONB         NOT NULL DEFAULT '[]',
    llm_params           JSONB         NOT NULL DEFAULT '{}',
    channel_format_rules JSONB         NOT NULL DEFAULT '{}',
    created_by           UUID,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    activated_at         TIMESTAMPTZ
);

CREATE INDEX idx_agent_configs_profile_status ON agent_configs (agent_profile_id, status);

-- Add FK from agent_profiles back to agent_configs now that the table exists
ALTER TABLE agent_profiles
    ADD CONSTRAINT fk_agent_profiles_active_config
    FOREIGN KEY (agent_config_id) REFERENCES agent_configs (id) ON DELETE SET NULL;
