-- +goose Up
-- Single-use, short-lived passwordless sign-in tokens. Only the SHA-256 hash of
-- the token is stored; the raw value exists only in the emailed link.
CREATE TABLE magic_link_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_public_id TEXT NOT NULL REFERENCES users (public_id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX magic_link_tokens_user_active_idx
    ON magic_link_tokens (user_public_id) WHERE used_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS magic_link_tokens;
