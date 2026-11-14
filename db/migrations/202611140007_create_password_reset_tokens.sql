-- +goose Up
-- Single-use, hashed tokens for the self-service password reset flow.
-- Admin invitations reuse this table, so the token is also the invite.
CREATE TABLE password_reset_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX password_reset_tokens_user_idx ON password_reset_tokens (user_id);

CREATE INDEX password_reset_tokens_pending_idx
    ON password_reset_tokens (token_hash) WHERE used_at IS NULL;

-- +goose Down
DROP TABLE password_reset_tokens;
