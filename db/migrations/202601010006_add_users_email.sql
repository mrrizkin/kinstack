-- +goose Up
ALTER TABLE users ADD COLUMN email TEXT NOT NULL DEFAULT '';

CREATE UNIQUE INDEX users_email_active_idx
    ON users (lower(email)) WHERE deleted_at IS NULL AND email <> '';

-- +goose Down
DROP INDEX IF EXISTS users_email_active_idx;
ALTER TABLE users DROP COLUMN IF EXISTS email;
