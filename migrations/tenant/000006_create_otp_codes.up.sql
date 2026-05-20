-- `users` lives in the `public` schema (global table). golang-migrate runs
-- this file with search_path=tenant_{slug}, so the unqualified reference
-- can't find `users`. Qualify it.
CREATE TABLE IF NOT EXISTS otp_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
    code_hash   TEXT        NOT NULL,
    used        BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_otp_codes_user_id ON otp_codes(user_id);
