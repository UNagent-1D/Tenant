CREATE TABLE IF NOT EXISTS end_users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    full_name     TEXT        NOT NULL,
    national_id   TEXT        UNIQUE,
    cellphone     TEXT        UNIQUE,
    email         TEXT,
    date_of_birth DATE,
    external_ref  TEXT,
    is_active     BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_end_users_national_id ON end_users (national_id);
CREATE INDEX idx_end_users_cellphone   ON end_users (cellphone);
