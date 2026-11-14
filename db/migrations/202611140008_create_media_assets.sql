-- +goose Up
CREATE TABLE media_assets (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    owner_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    object_key TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX media_assets_owner_active_idx
    ON media_assets (owner_id) WHERE deleted_at IS NULL;

INSERT INTO role_permissions (role_code, permission_code) VALUES
    ('administrator', 'media.view'),
    ('administrator', 'media.manage');

-- +goose Down
DELETE FROM role_permissions
WHERE role_code = 'administrator'
  AND permission_code IN ('media.view', 'media.manage');
DROP TABLE media_assets;
