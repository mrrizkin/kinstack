-- +goose Up
-- Personal API tokens are long-lived bearer credentials. Only the SHA-256 hash
-- of the secret portion is stored; the plaintext is shown once at creation.
CREATE TABLE api_tokens (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    user_public_id TEXT NOT NULL REFERENCES users (public_id) ON DELETE CASCADE,
    name TEXT NOT NULL DEFAULT '',
    token_hash TEXT NOT NULL,
    prefix TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX api_tokens_user_idx
    ON api_tokens (user_public_id) WHERE revoked_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS api_tokens;
