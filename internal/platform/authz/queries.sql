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
SELECT up.permission_code, up.effect, up.reason, up.expires_at
FROM user_permissions up
JOIN users u ON u.id = up.user_id
WHERE u.public_id = sqlc.arg('user_public_id')
ORDER BY up.permission_code ASC;

-- name: UserOverridesMany :many
SELECT u.public_id AS user_public_id, up.permission_code, up.effect, up.reason, up.expires_at
FROM user_permissions up
JOIN users u ON u.id = up.user_id
WHERE u.public_id = ANY(sqlc.arg('user_public_ids')::text[])
ORDER BY u.public_id ASC, up.permission_code ASC;

-- name: UpsertUserOverride :one
INSERT INTO user_permissions (user_id, permission_code, effect, reason, expires_at)
SELECT
    u.id,
    sqlc.arg('permission_code'),
    sqlc.arg('effect'),
    sqlc.arg('reason'),
    sqlc.arg('expires_at')
FROM users u
WHERE u.public_id = sqlc.arg('user_public_id')
ON CONFLICT (user_id, permission_code)
DO UPDATE SET effect = EXCLUDED.effect,
              reason = EXCLUDED.reason,
              expires_at = EXCLUDED.expires_at,
              updated_at = now()
RETURNING permission_code, effect, reason, expires_at;

-- name: DeleteUserOverride :exec
DELETE FROM user_permissions up
WHERE up.user_id = (
        SELECT u.id FROM users u
        WHERE u.public_id = sqlc.arg('user_public_id')
    )
  AND up.permission_code = sqlc.arg('permission_code');

-- name: AssignUserRole :execrows
WITH deleted AS (
    DELETE FROM user_roles
    WHERE user_id = (
        SELECT u.id FROM users u
        WHERE u.public_id = sqlc.arg('user_public_id')
    )
)
INSERT INTO user_roles (user_id, role_code)
SELECT u.id, sqlc.arg('role_code')
FROM users u
WHERE u.public_id = sqlc.arg('user_public_id')
  AND u.deleted_at IS NULL
RETURNING user_id;

-- name: UserRole :one
SELECT ur.role_code
FROM user_roles ur
JOIN users u ON u.id = ur.user_id
WHERE u.public_id = sqlc.arg('user_public_id')
LIMIT 1;

-- name: CountRoleAssignments :one
SELECT count(DISTINCT ur.user_id)
FROM user_roles ur
JOIN users u ON u.id = ur.user_id
WHERE ur.role_code = sqlc.arg('role_code')
  AND u.deleted_at IS NULL;
