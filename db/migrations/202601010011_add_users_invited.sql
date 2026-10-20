-- +goose Up
-- invited_at marks an account created through an admin invitation that has not
-- set its own password yet. The password hash is unusable until then.
ALTER TABLE users ADD COLUMN invited_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE users DROP COLUMN invited_at;
