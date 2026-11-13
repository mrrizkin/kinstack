-- name: FindUserByUsername :one
SELECT
    u.public_id,
    u.username,
    u.email,
    u.display_name,
    u.password_hash,
    COALESCE((
        SELECT ur.role_code
        FROM user_roles ur
        JOIN roles r ON r.code = ur.role_code
        WHERE ur.user_public_id = u.public_id
        ORDER BY r.rank DESC, ur.role_code ASC
        LIMIT 1
    ), '')::text AS role,
    u.is_active
FROM users u
WHERE lower(u.username) = lower(sqlc.arg('username'))
  AND u.deleted_at IS NULL
LIMIT 1;

-- name: FindAuthUserByPublicID :one
SELECT
    u.public_id,
    u.username,
    u.email,
    u.display_name,
    u.password_hash,
    COALESCE((
        SELECT ur.role_code
        FROM user_roles ur
        JOIN roles r ON r.code = ur.role_code
        WHERE ur.user_public_id = u.public_id
        ORDER BY r.rank DESC, ur.role_code ASC
        LIMIT 1
    ), '')::text AS role,
    u.is_active
FROM users u
WHERE u.public_id = sqlc.arg('public_id')
  AND u.deleted_at IS NULL
LIMIT 1;

-- name: FindUserByEmail :one
SELECT
    u.public_id,
    u.username,
    u.email,
    u.display_name,
    u.password_hash,
    COALESCE((
        SELECT ur.role_code
        FROM user_roles ur
        JOIN roles r ON r.code = ur.role_code
        WHERE ur.user_public_id = u.public_id
        ORDER BY r.rank DESC, ur.role_code ASC
        LIMIT 1
    ), '')::text AS role,
    u.is_active
FROM users u
WHERE lower(u.email) = lower(sqlc.arg('email'))
  AND u.email <> ''
  AND u.deleted_at IS NULL
LIMIT 1;

-- name: FindIdentityUser :one
SELECT
    u.public_id,
    u.username,
    u.email,
    u.display_name,
    COALESCE(u.password_hash, '')::text AS password_hash,
    COALESCE((
        SELECT ur.role_code
        FROM user_roles ur
        JOIN roles r ON r.code = ur.role_code
        WHERE ur.user_public_id = u.public_id
        ORDER BY r.rank DESC, ur.role_code ASC
        LIMIT 1
    ), '')::text AS role,
    u.is_active
FROM user_identities i
JOIN users u ON u.id = i.user_id
WHERE i.provider = sqlc.arg('provider')
  AND i.subject = sqlc.arg('subject')
  AND u.deleted_at IS NULL
LIMIT 1;

-- name: UpsertIdentity :execrows
INSERT INTO user_identities (user_id, provider, subject, email)
SELECT u.id, sqlc.arg('provider'), sqlc.arg('subject'), sqlc.arg('email')
FROM users u
WHERE u.public_id = sqlc.arg('user_public_id')
  AND u.deleted_at IS NULL
ON CONFLICT (provider, subject)
DO UPDATE SET email = EXCLUDED.email, updated_at = now()
WHERE user_identities.user_id = EXCLUDED.user_id;

-- name: TouchLastLogin :exec
UPDATE users
SET last_login_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL;

-- name: CreatePasswordResetToken :exec
INSERT INTO password_reset_tokens (user_public_id, token_hash, expires_at)
VALUES (
    sqlc.arg('user_public_id'),
    sqlc.arg('token_hash'),
    sqlc.arg('expires_at')
);

-- name: ConsumePasswordResetToken :one
UPDATE password_reset_tokens
SET used_at = now()
WHERE token_hash = sqlc.arg('token_hash')
  AND used_at IS NULL
  AND expires_at > now()
RETURNING user_public_id;

-- name: UpdatePasswordHash :execrows
UPDATE users
SET password_hash = sqlc.arg('password_hash'),
    invited_at = NULL,
    updated_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL;
