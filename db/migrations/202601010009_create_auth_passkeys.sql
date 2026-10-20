-- +goose Up
-- WebAuthn credentials for passkey sign-in. The full credential JSON is stored
-- so authenticator counter/signature state survives verification.
CREATE TABLE auth_passkeys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    credential_id BYTEA NOT NULL UNIQUE,
    name TEXT NOT NULL DEFAULT '',
    credential JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ
);

CREATE INDEX auth_passkeys_user_idx ON auth_passkeys (user_id);

-- +goose Down
DROP TABLE IF EXISTS auth_passkeys;
