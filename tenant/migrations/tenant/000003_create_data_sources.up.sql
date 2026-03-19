CREATE TYPE source_type AS ENUM ('scheduling', 'patient_registry');

CREATE TABLE IF NOT EXISTS data_sources (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name           TEXT        NOT NULL,
    source_type    source_type NOT NULL,
    base_url       TEXT        NOT NULL,
    credential_ref TEXT,
    route_configs  JSONB       NOT NULL DEFAULT '{}',
    is_active      BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
