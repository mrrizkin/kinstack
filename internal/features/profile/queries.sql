-- name: FindProfile :one
SELECT
    u.public_id,
    u.username,
    u.display_name,
    COALESCE((
        SELECT ur.role_code
        FROM user_roles ur
        JOIN roles r ON r.code = ur.role_code
        WHERE ur.user_id = u.id
        ORDER BY r.rank DESC, ur.role_code ASC
        LIMIT 1
    ), '')::text AS role,
    COALESCE(p.theme_preference, 'system') AS theme_preference,
    COALESCE(p.sound_enabled, true) AS sound_enabled,
    COALESCE(p.page_size, 5) AS page_size,
    (u.invited_at IS NOT NULL)::boolean AS invited,
    u.last_login_at,
    u.created_at
FROM users u
LEFT JOIN user_preferences p ON p.user_id = u.id
WHERE u.public_id = sqlc.arg('public_id')
  AND u.deleted_at IS NULL
LIMIT 1;

-- name: FindPasswordHash :one
SELECT password_hash
FROM users
WHERE public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL
LIMIT 1;

-- name: FindPageSize :one
SELECT COALESCE(p.page_size, 5) AS page_size
FROM users u
LEFT JOIN user_preferences p ON p.user_id = u.id
WHERE u.public_id = sqlc.arg('public_id')
  AND u.deleted_at IS NULL
LIMIT 1;

-- name: UpdateDisplayName :one
WITH updated AS (
    UPDATE users
    SET display_name = sqlc.arg('display_name'),
        updated_at = now()
    WHERE public_id = sqlc.arg('public_id')
      AND deleted_at IS NULL
    RETURNING *
)
SELECT
    u.public_id,
    u.username,
    u.display_name,
    COALESCE((
        SELECT ur.role_code
        FROM user_roles ur
        JOIN roles r ON r.code = ur.role_code
        WHERE ur.user_id = u.id
        ORDER BY r.rank DESC, ur.role_code ASC
        LIMIT 1
    ), '')::text AS role,
    COALESCE(p.theme_preference, 'system') AS theme_preference,
    COALESCE(p.sound_enabled, true) AS sound_enabled,
    COALESCE(p.page_size, 5) AS page_size,
    (u.invited_at IS NOT NULL)::boolean AS invited,
    u.last_login_at,
    u.created_at
FROM updated u
LEFT JOIN user_preferences p ON p.user_id = u.id;

-- name: UpdatePreferences :one
WITH target AS (
    SELECT id
    FROM users
    WHERE public_id = sqlc.arg('public_id')
      AND deleted_at IS NULL
)
INSERT INTO user_preferences (user_id, theme_preference, sound_enabled, page_size, updated_at)
SELECT id,
       sqlc.arg('theme_preference'),
       sqlc.arg('sound_enabled'),
       sqlc.arg('page_size'),
       now()
FROM target
ON CONFLICT (user_id) DO UPDATE
SET theme_preference = EXCLUDED.theme_preference,
    sound_enabled = EXCLUDED.sound_enabled,
    page_size = EXCLUDED.page_size,
    updated_at = now()
RETURNING theme_preference, sound_enabled, page_size;

-- name: MarkPasswordSet :execrows
UPDATE users
SET password_hash = sqlc.arg('password_hash'),
    invited_at = NULL,
    updated_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL;
