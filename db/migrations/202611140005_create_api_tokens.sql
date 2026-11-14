-- +goose Up
-- Personal API tokens are long-lived bearer credentials. Only the SHA-256 hash
-- of the secret portion is stored; the plaintext is shown once at creation.
-- Scopes are an allowlist of permission codes: the effective permission of a
-- bearer request is the intersection of the owner's permissions and these
-- scopes, and an empty array grants nothing.
CREATE TABLE api_tokens (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT '',
    token_hash TEXT NOT NULL,
    prefix TEXT NOT NULL DEFAULT '',
    scopes TEXT[] NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX api_tokens_user_active_idx
    ON api_tokens (user_id) WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE api_tokens;
