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
    role_code       TEXT NOT NULL REFERENCES roles(code) ON DELETE CASCADE,
    permission_code TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_code, permission_code)
);

CREATE TABLE user_roles (
    user_public_id TEXT NOT NULL REFERENCES users (public_id) ON DELETE CASCADE,
    role_code      TEXT NOT NULL REFERENCES roles(code) ON DELETE CASCADE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_public_id, role_code)
);

CREATE INDEX user_roles_role_idx ON user_roles (role_code);

CREATE TABLE user_permissions (
    user_public_id  TEXT NOT NULL,
    permission_code TEXT NOT NULL,
    effect          TEXT NOT NULL CHECK (effect IN ('allow', 'deny')),
    reason          TEXT NOT NULL DEFAULT '',
    expires_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_public_id, permission_code)
);

CREATE INDEX user_permissions_user_idx ON user_permissions (user_public_id);

-- +goose Down
DROP TABLE IF EXISTS user_permissions;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS roles;
