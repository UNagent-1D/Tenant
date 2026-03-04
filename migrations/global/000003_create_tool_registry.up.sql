CREATE TABLE IF NOT EXISTS tool_registry (
    id                  UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    name                TEXT    NOT NULL UNIQUE,
    description         TEXT,
    openai_function_def JSONB   NOT NULL DEFAULT '{}',
    is_active           BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tool_registry_name      ON tool_registry (name);
CREATE INDEX idx_tool_registry_is_active ON tool_registry (is_active);
