-- +goose Up
-- TOTP secrets are sealed with an application-held key instead of being stored
-- as base32 text, so a database dump alone cannot mint valid codes. Existing
-- rows are migrated lazily on the next successful verification (or eagerly with
-- the two-factor backfill command); the plaintext column stays until then and is
-- cleared as soon as a sealed copy exists.
ALTER TABLE users
    ADD COLUMN two_factor_secret_enc BYTEA NOT NULL DEFAULT ''::bytea,
    ADD COLUMN two_factor_key_version INTEGER NOT NULL DEFAULT 0,
    -- Last accepted TOTP timestep, used to reject replayed codes.
    ADD COLUMN two_factor_last_step BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE users
    DROP COLUMN two_factor_last_step,
    DROP COLUMN two_factor_key_version,
    DROP COLUMN two_factor_secret_enc;
