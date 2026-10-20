-- +goose Up
-- Grant the audit viewer permission to the seeded administrator role, following
-- the convention that each feature slice seeds its own permissions.
INSERT INTO role_permissions (role_code, permission_code) VALUES
    ('administrator', 'audit.view');

-- +goose Down
DELETE FROM role_permissions
WHERE role_code = 'administrator' AND permission_code = 'audit.view';
