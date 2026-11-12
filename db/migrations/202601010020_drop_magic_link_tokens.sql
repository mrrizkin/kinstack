-- +goose Up
-- Passwordless sign-in is gone. Invitations reuse password_reset_tokens, so the
-- dedicated token table has no remaining reader.
DROP TABLE IF EXISTS magic_link_tokens;

-- +goose Down
-- Restore the table so the pair round-trips on a database that still has it.
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
