-- name: CreateToken :one
WITH inserted AS (
    INSERT INTO api_tokens (public_id, user_id, name, token_hash, prefix, scopes, expires_at)
    SELECT
        sqlc.arg('public_id'),
        u.id,
        sqlc.arg('name'),
        sqlc.arg('token_hash'),
        sqlc.arg('prefix'),
        sqlc.arg('scopes'),
        sqlc.arg('expires_at')
    FROM users u
    WHERE u.public_id = sqlc.arg('user_public_id')
      AND u.deleted_at IS NULL
    RETURNING *
)
SELECT
    t.public_id,
    u.public_id AS user_public_id,
    t.name,
    t.token_hash,
    t.prefix,
    t.scopes,
    t.expires_at,
    t.last_used_at,
    t.revoked_at,
    t.created_at
FROM inserted t
JOIN users u ON u.id = t.user_id;

-- name: ListTokensByUser :many
SELECT
    t.public_id,
    u.public_id AS user_public_id,
    t.name,
    t.token_hash,
    t.prefix,
    t.scopes,
    t.expires_at,
    t.last_used_at,
    t.revoked_at,
    t.created_at
FROM api_tokens t
JOIN users u ON u.id = t.user_id
WHERE u.public_id = sqlc.arg('user_public_id')
  AND t.revoked_at IS NULL
ORDER BY t.created_at DESC, t.id DESC
LIMIT sqlc.arg('row_limit') OFFSET sqlc.arg('row_offset');

-- name: CountTokensByUser :one
SELECT count(*)
FROM api_tokens t
JOIN users u ON u.id = t.user_id
WHERE u.public_id = sqlc.arg('user_public_id')
  AND t.revoked_at IS NULL;

-- name: FindTokenByPublicID :one
SELECT
    t.public_id,
    u.public_id AS user_public_id,
    t.name,
    t.token_hash,
    t.prefix,
    t.scopes,
    t.expires_at,
    t.last_used_at,
    t.revoked_at,
    t.created_at
FROM api_tokens t
JOIN users u ON u.id = t.user_id
WHERE t.public_id = sqlc.arg('public_id')
LIMIT 1;

-- name: RevokeToken :execrows
UPDATE api_tokens
SET revoked_at = now()
WHERE api_tokens.public_id = sqlc.arg('public_id')
  AND api_tokens.user_id = (
      SELECT u.id FROM users u
      WHERE u.public_id = sqlc.arg('user_public_id')
  )
  AND revoked_at IS NULL;

-- name: RevokeTokensByUser :execrows
UPDATE api_tokens
SET revoked_at = now()
WHERE api_tokens.user_id = (
    SELECT u.id FROM users u
    WHERE u.public_id = sqlc.arg('user_public_id')
)
  AND api_tokens.revoked_at IS NULL;

-- name: TouchTokenLastUsed :exec
UPDATE api_tokens
SET last_used_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND revoked_at IS NULL;
