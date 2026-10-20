-- +goose Up
-- Grant the media viewer to the seeded administrator role.
INSERT INTO role_permissions (role_code, permission_code) VALUES
    ('administrator', 'media.view'),
    ('administrator', 'media.manage');

-- +goose Down
DELETE FROM role_permissions
WHERE role_code = 'administrator'
  AND permission_code IN ('media.view', 'media.manage');
