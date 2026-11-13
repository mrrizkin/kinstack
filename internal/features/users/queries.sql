-- Read queries for the users slice. Generated as type-checked functions; the
-- repository keeps the write path in sqlx.

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
LIMIT $1 OFFSET $2;

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
WHERE u.public_id = $1
  AND u.deleted_at IS NULL
LIMIT 1;
