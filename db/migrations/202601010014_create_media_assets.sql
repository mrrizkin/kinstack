-- +goose Up
CREATE TABLE media_assets (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    owner_public_id TEXT NOT NULL REFERENCES users (public_id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    object_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX media_assets_owner_active_idx
    ON media_assets (owner_public_id) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS media_assets;
