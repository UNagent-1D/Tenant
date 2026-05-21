-- Optional first/last name on users. NULL for legacy rows; the UI accepts
-- empty strings and the JWT doesn't carry them.
ALTER TABLE users ADD COLUMN first_name TEXT;
ALTER TABLE users ADD COLUMN last_name  TEXT;
