-- +goose Up
-- Identity. `public_id` is the external handle used by routes, sessions, and
-- API clients; every reference inside the database uses `id` instead.
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    username TEXT NOT NULL,
    display_name TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    -- TOTP secrets are sealed with an application-held key. The plaintext
    -- column is retained only so the lazy backfill can find pre-seal rows; new
    -- secrets are written to two_factor_secret_enc.
    two_factor_secret TEXT NOT NULL DEFAULT '',
    two_factor_secret_enc BYTEA NOT NULL DEFAULT ''::bytea,
    two_factor_key_version INTEGER NOT NULL DEFAULT 0,
    -- Last accepted TOTP timestep, used to reject replayed codes.
    two_factor_last_step BIGINT NOT NULL DEFAULT 0,
    two_factor_enabled BOOLEAN NOT NULL DEFAULT false,
    two_factor_confirmed_at TIMESTAMPTZ,
    email TEXT NOT NULL DEFAULT '',
    -- invited_at marks an account created through an admin invitation that has
    -- not set its own password yet.
    invited_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX users_username_active_idx
    ON users (lower(username)) WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX users_email_active_idx
    ON users (lower(email)) WHERE deleted_at IS NULL AND email <> '';

CREATE TABLE user_sessions (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    session_id TEXT NOT NULL UNIQUE,
    ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ
);

CREATE INDEX user_sessions_user_active_idx
    ON user_sessions (user_id) WHERE revoked_at IS NULL;

CREATE TABLE user_recovery_codes (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX user_recovery_codes_user_idx ON user_recovery_codes (user_id);

-- +goose Down
DROP TABLE user_recovery_codes;
DROP TABLE user_sessions;
DROP TABLE users;
