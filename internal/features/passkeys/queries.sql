-- name: FindPasskeyUserByPublicID :one
SELECT public_id, email, username, display_name, is_active, deleted_at
FROM users
WHERE public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL
  AND is_active
LIMIT 1;

-- name: ListPasskeys :many
SELECT p.credential_id, p.name, p.credential, p.created_at, p.last_used_at
FROM auth_passkeys p
JOIN users u ON u.id = p.user_id
WHERE u.public_id = sqlc.arg('public_id')
  AND u.deleted_at IS NULL
  AND u.is_active
ORDER BY p.created_at ASC, p.id ASC;

-- name: SavePasskey :execrows
INSERT INTO auth_passkeys (user_id, credential_id, name, credential)
SELECT u.id,
       sqlc.arg('credential_id'),
       sqlc.arg('name'),
       sqlc.arg('credential')::jsonb
FROM users u
WHERE u.public_id = sqlc.arg('public_id')
  AND u.deleted_at IS NULL
  AND u.is_active;

-- name: UpdatePasskey :execrows
UPDATE auth_passkeys p
SET credential = sqlc.arg('credential')::jsonb,
    last_used_at = COALESCE(sqlc.arg('last_used_at'), p.last_used_at)
FROM users u
WHERE p.user_id = u.id
  AND u.public_id = sqlc.arg('public_id')
  AND p.credential_id = sqlc.arg('credential_id')
  AND u.deleted_at IS NULL
  AND u.is_active;

-- name: DeletePasskey :execrows
DELETE FROM auth_passkeys p
USING users u
WHERE p.user_id = u.id
  AND u.public_id = sqlc.arg('public_id')
  AND p.credential_id = sqlc.arg('credential_id')
  AND u.deleted_at IS NULL
  AND u.is_active;

-- name: FindPasskeyLoginUser :one
SELECT u.public_id, u.email, u.username, u.display_name, u.is_active, u.deleted_at
FROM auth_passkeys p
JOIN users u ON u.id = p.user_id
WHERE p.credential_id = sqlc.arg('credential_id')
  AND u.deleted_at IS NULL
  AND u.is_active
LIMIT 1;

-- name: HasActivePasskeyUser :one
SELECT EXISTS (
    SELECT 1 FROM users
    WHERE public_id = sqlc.arg('public_id') AND deleted_at IS NULL AND is_active
);
