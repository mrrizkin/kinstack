-- +goose Up
CREATE TABLE audit_events (
    id BIGSERIAL PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    request_id TEXT NOT NULL DEFAULT '',
    actor_key TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    resource TEXT NOT NULL DEFAULT '',
    method TEXT NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    outcome TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX audit_events_occurred_idx ON audit_events (occurred_at DESC, id DESC);
CREATE INDEX audit_events_actor_idx ON audit_events (actor_key, occurred_at DESC);

-- Feature slices seed their own permissions onto the built-in admin role.
INSERT INTO role_permissions (role_code, permission_code) VALUES
    ('administrator', 'audit.view');

-- +goose Down
DELETE FROM role_permissions
WHERE role_code = 'administrator' AND permission_code = 'audit.view';
DROP TABLE audit_events;
