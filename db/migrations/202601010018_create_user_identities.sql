-- +goose Up
-- External identities are bound to (provider, subject), not to an email
-- address. Email ownership can change at the provider; the subject is stable.
CREATE TABLE user_identities (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider TEXT NOT NULL,
    subject TEXT NOT NULL,
    email TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_identities_provider_subject_key UNIQUE (provider, subject)
);

CREATE INDEX user_identities_user_idx ON user_identities (user_id);

-- +goose Down
DROP TABLE IF EXISTS user_identities;
