-- name: ListRoles :many
SELECT code, name, description, rank, is_system, created_at, updated_at
FROM roles
ORDER BY rank DESC, code ASC
LIMIT sqlc.arg('row_limit') OFFSET sqlc.arg('row_offset');

-- name: CountRoles :one
SELECT COUNT(*) FROM roles;

-- name: FindRole :one
SELECT code, name, description, rank, is_system, created_at, updated_at
FROM roles
WHERE code = sqlc.arg('code')
LIMIT 1;

-- name: CreateRole :one
INSERT INTO roles (code, name, description, rank)
VALUES (
    sqlc.arg('code'),
    sqlc.arg('name'),
    sqlc.arg('description'),
    sqlc.arg('rank')
)
RETURNING code, name, description, rank, is_system, created_at, updated_at;

-- name: UpdateRole :one
UPDATE roles
SET name = sqlc.arg('name'),
    description = sqlc.arg('description'),
    rank = sqlc.arg('rank'),
    updated_at = now()
WHERE code = sqlc.arg('code')
  AND is_system = false
RETURNING code, name, description, rank, is_system, created_at, updated_at;

-- name: DeleteRole :execrows
DELETE FROM roles
WHERE code = sqlc.arg('code')
  AND is_system = false;

-- name: RolePermissions :many
SELECT permission_code
FROM role_permissions
WHERE role_code = sqlc.arg('role_code')
ORDER BY permission_code ASC;

-- name: RolePermissionsMany :many
SELECT role_code, permission_code
FROM role_permissions
WHERE role_code = ANY(sqlc.arg('role_codes')::text[])
ORDER BY role_code ASC, permission_code ASC;

-- name: ReplaceRolePermissions :exec
WITH target AS (
    SELECT code FROM roles
    WHERE code = sqlc.arg('code')
      AND is_system = false
),
deleted AS (
    DELETE FROM role_permissions
    WHERE role_code IN (SELECT code FROM target)
)
INSERT INTO role_permissions (role_code, permission_code)
SELECT target.code, unnest(sqlc.arg('permission_codes')::text[])
FROM target;

-- name: UserOverrides :many
SELECT permission_code, effect, reason, expires_at
FROM user_permissions
WHERE user_public_id = sqlc.arg('user_public_id')
ORDER BY permission_code ASC;

-- name: UserOverridesMany :many
SELECT user_public_id, permission_code, effect, reason, expires_at
FROM user_permissions
WHERE user_public_id = ANY(sqlc.arg('user_public_ids')::text[])
ORDER BY user_public_id ASC, permission_code ASC;

-- name: UpsertUserOverride :one
INSERT INTO user_permissions (user_public_id, permission_code, effect, reason, expires_at)
VALUES (
    sqlc.arg('user_public_id'),
    sqlc.arg('permission_code'),
    sqlc.arg('effect'),
    sqlc.arg('reason'),
    sqlc.arg('expires_at')
)
ON CONFLICT (user_public_id, permission_code)
DO UPDATE SET effect = EXCLUDED.effect,
              reason = EXCLUDED.reason,
              expires_at = EXCLUDED.expires_at,
              updated_at = now()
RETURNING permission_code, effect, reason, expires_at;

-- name: DeleteUserOverride :exec
DELETE FROM user_permissions
WHERE user_public_id = sqlc.arg('user_public_id')
  AND permission_code = sqlc.arg('permission_code');

-- name: AssignUserRole :execrows
WITH deleted AS (
    DELETE FROM user_roles WHERE user_public_id = sqlc.arg('user_public_id')
)
INSERT INTO user_roles (user_public_id, role_code)
SELECT sqlc.arg('user_public_id'), sqlc.arg('role_code')
WHERE EXISTS (
    SELECT 1 FROM users
    WHERE public_id = sqlc.arg('user_public_id')
      AND deleted_at IS NULL
)
RETURNING user_public_id;

-- name: UserRole :one
SELECT role_code
FROM user_roles
WHERE user_public_id = sqlc.arg('user_public_id')
LIMIT 1;

-- name: CountRoleAssignments :one
SELECT count(DISTINCT ur.user_public_id)
FROM user_roles ur
JOIN users u ON u.public_id = ur.user_public_id
WHERE ur.role_code = sqlc.arg('role_code')
  AND u.deleted_at IS NULL;
