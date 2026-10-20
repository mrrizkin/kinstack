-- +goose Up
-- Seed one system role operators can assign to their first real admin account.
-- System roles are immutable: the access UI cannot rename them, re-rank them,
-- delete them, or replace their permissions. Feature slices grant their new
-- permissions to this role in their own seed migration; changing the built-in
-- set therefore requires a migration, not a runtime edit.
INSERT INTO roles (code, name, description, rank, is_system) VALUES
    ('administrator', 'Administrator', 'Full system administration access', 100, true);

INSERT INTO role_permissions (role_code, permission_code) VALUES
    ('administrator', 'users.view'),
    ('administrator', 'users.manage'),
    ('administrator', 'access.manage');

-- +goose Down
DELETE FROM role_permissions WHERE role_code = 'administrator';
DELETE FROM roles WHERE code = 'administrator';
