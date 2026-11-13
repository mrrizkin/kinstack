-- Queries for the users slice. Generated as type-checked functions; the
-- repository keeps transactions, advisory locking, and error classification.

-- name: ListUsers :many
SELECT
    u.public_id,
    u.username,
    u.email,
    u.display_name,
    COALESCE((
        SELECT ur.role_code
        FROM user_roles ur
        JOIN roles r ON r.code = ur.role_code
        WHERE ur.user_public_id = u.public_id
        ORDER BY r.rank DESC, ur.role_code ASC
        LIMIT 1
    ), '')::text AS role,
    u.is_active,
    (u.invited_at IS NOT NULL)::boolean AS invited,
    u.last_login_at,
    u.created_at
FROM users u
WHERE u.deleted_at IS NULL
ORDER BY u.created_at ASC, u.id ASC
LIMIT sqlc.arg('row_limit') OFFSET sqlc.arg('row_offset');

-- name: CountUsers :one
SELECT
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE u.is_active) AS active
FROM users u
WHERE u.deleted_at IS NULL;

-- name: FindUserByPublicID :one
SELECT
    u.public_id,
    u.username,
    u.email,
    u.display_name,
    COALESCE((
        SELECT ur.role_code
        FROM user_roles ur
        JOIN roles r ON r.code = ur.role_code
        WHERE ur.user_public_id = u.public_id
        ORDER BY r.rank DESC, ur.role_code ASC
        LIMIT 1
    ), '')::text AS role,
    u.is_active,
    (u.invited_at IS NOT NULL)::boolean AS invited,
    u.last_login_at,
    u.created_at
FROM users u
WHERE u.public_id = sqlc.arg('public_id')
  AND u.deleted_at IS NULL
LIMIT 1;

-- name: CreateUser :one
WITH inserted AS (
    INSERT INTO users (public_id, username, email, display_name, password_hash, invited_at)
    VALUES (
        sqlc.arg('public_id'),
        sqlc.arg('username'),
        sqlc.arg('email'),
        sqlc.arg('display_name'),
        sqlc.arg('password_hash'),
        CASE WHEN sqlc.arg('invited')::boolean THEN now() ELSE NULL END
    )
    RETURNING *
)
SELECT
    u.public_id,
    u.username,
    u.email,
    u.display_name,
    COALESCE((
        SELECT ur.role_code
        FROM user_roles ur
        JOIN roles r ON r.code = ur.role_code
        WHERE ur.user_public_id = u.public_id
        ORDER BY r.rank DESC, ur.role_code ASC
        LIMIT 1
    ), '')::text AS role,
    u.is_active,
    (u.invited_at IS NOT NULL)::boolean AS invited,
    u.last_login_at,
    u.created_at
FROM inserted u;

-- name: UpdateUser :one
WITH updated AS (
    UPDATE users
    SET display_name = sqlc.arg('display_name'),
        email = sqlc.arg('email'),
        updated_at = now()
    WHERE public_id = sqlc.arg('public_id')
      AND deleted_at IS NULL
    RETURNING *
)
SELECT
    u.public_id,
    u.username,
    u.email,
    u.display_name,
    COALESCE((
        SELECT ur.role_code
        FROM user_roles ur
        JOIN roles r ON r.code = ur.role_code
        WHERE ur.user_public_id = u.public_id
        ORDER BY r.rank DESC, ur.role_code ASC
        LIMIT 1
    ), '')::text AS role,
    u.is_active,
    (u.invited_at IS NOT NULL)::boolean AS invited,
    u.last_login_at,
    u.created_at
FROM updated u;

-- name: UpdateUserPassword :execrows
UPDATE users
SET password_hash = sqlc.arg('password_hash'),
    updated_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL;

-- name: SetUserActive :one
WITH updated AS (
    UPDATE users
    SET is_active = sqlc.arg('is_active'),
        updated_at = now()
    WHERE public_id = sqlc.arg('public_id')
      AND deleted_at IS NULL
    RETURNING *
)
SELECT
    u.public_id,
    u.username,
    u.email,
    u.display_name,
    COALESCE((
        SELECT ur.role_code
        FROM user_roles ur
        JOIN roles r ON r.code = ur.role_code
        WHERE ur.user_public_id = u.public_id
        ORDER BY r.rank DESC, ur.role_code ASC
        LIMIT 1
    ), '')::text AS role,
    u.is_active,
    (u.invited_at IS NOT NULL)::boolean AS invited,
    u.last_login_at,
    u.created_at
FROM updated u;

-- name: LockAccessManagers :exec
-- Serializes every decision that could remove the last account able to manage
-- access. The key is a constant, so all replicas contend on the same advisory
-- lock.
SELECT pg_advisory_xact_lock(hashtext('kinstack:access_managers'));

-- name: CountActiveAccessManagers :one
SELECT COUNT(*)
FROM users u
WHERE u.public_id <> sqlc.arg('public_id')
  AND u.is_active
  AND u.deleted_at IS NULL
  AND (
    EXISTS (
        SELECT 1
        FROM user_roles ur
        JOIN role_permissions rp ON rp.role_code = ur.role_code
        WHERE ur.user_public_id = u.public_id
          AND rp.permission_code = 'access.manage'
      )
    OR EXISTS (
        SELECT 1
        FROM user_permissions up
        WHERE up.user_public_id = u.public_id
          AND up.permission_code = 'access.manage'
          AND up.effect = 'allow'
          AND (up.expires_at IS NULL OR up.expires_at > now())
      )
  )
  AND NOT EXISTS (
    SELECT 1
    FROM user_permissions up
    WHERE up.user_public_id = u.public_id
      AND up.permission_code = 'access.manage'
      AND up.effect = 'deny'
      AND (up.expires_at IS NULL OR up.expires_at > now())
  );
