CREATE TABLE end_users (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL,
    full_name    TEXT NOT NULL,
    national_id  TEXT NOT NULL,
    cellphone    TEXT NOT NULL,
    email        TEXT,
    date_of_birth TEXT,
    external_ref TEXT,
    is_active    BOOLEAN NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(tenant_id, national_id),
    UNIQUE(tenant_id, cellphone)
);

CREATE INDEX idx_end_users_cellphone    ON end_users(tenant_id, cellphone);
CREATE INDEX idx_end_users_national_id  ON end_users(tenant_id, national_id);
