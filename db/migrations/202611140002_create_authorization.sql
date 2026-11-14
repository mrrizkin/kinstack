-- +goose Up
-- Hybrid RBAC + ABAC authorization.
--
-- RBAC baseline: roles own a set of permission codes and a user may hold one or
-- more roles through user_roles.
-- Fine-grained override: a user may receive an explicit allow/deny for a single
-- permission (deny wins, optional expiry) without changing their role.
-- ABAC guardrails are code-defined policies in internal/platform/authz that can
-- read request attributes (ownership, amount, time, shift state).
--
-- Roles are data, owned by the database and managed from the access UI. No role
-- name is ever referenced by application code; the engine resolves permissions
-- purely through role_permissions and user_permissions.
-- Permission codes are owned by the Go catalog (internal/platform/authz/catalog.go);
-- role_permissions/user_permissions codes are validated against that catalog at the
-- service boundary.
CREATE TABLE roles (
    code        TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    rank        INTEGER NOT NULL DEFAULT 0,
    is_system   BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE role_permissions (
    role_code       TEXT NOT NULL REFERENCES roles (code) ON DELETE CASCADE,
    permission_code TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_code, permission_code)
);

CREATE TABLE user_roles (
    user_id    BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_code  TEXT NOT NULL REFERENCES roles (code) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, role_code)
);

CREATE INDEX user_roles_role_idx ON user_roles (role_code);

CREATE TABLE user_permissions (
    user_id         BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    permission_code TEXT NOT NULL,
    effect          TEXT NOT NULL CHECK (effect IN ('allow', 'deny')),
    reason          TEXT NOT NULL DEFAULT '',
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, permission_code)
);

-- Seed one system role operators can assign to their first real admin account.
-- System roles are immutable: the access UI cannot rename them, re-rank them,
-- delete them, or replace their permissions. Feature slices grant their new
-- permissions to this role in their own migration; changing the built-in set
-- therefore requires a migration, not a runtime edit.
INSERT INTO roles (code, name, description, rank, is_system) VALUES
    ('administrator', 'Administrator', 'Full system administration access', 100, true);

INSERT INTO role_permissions (role_code, permission_code) VALUES
    ('administrator', 'users.view'),
    ('administrator', 'users.manage'),
    ('administrator', 'access.manage');

-- +goose Down
DROP TABLE user_permissions;
DROP TABLE user_roles;
DROP TABLE role_permissions;
DROP TABLE roles;
