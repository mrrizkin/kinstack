-- name: CreateToken :one
INSERT INTO api_tokens (public_id, user_public_id, name, token_hash, prefix, scopes, expires_at)
VALUES (
    sqlc.arg('public_id'),
    sqlc.arg('user_public_id'),
    sqlc.arg('name'),
    sqlc.arg('token_hash'),
    sqlc.arg('prefix'),
    sqlc.arg('scopes'),
    sqlc.arg('expires_at')
)
RETURNING
    public_id,
    user_public_id,
    name,
    token_hash,
    prefix,
    scopes,
    expires_at,
    last_used_at,
    revoked_at,
    created_at;

-- name: ListTokensByUser :many
SELECT
    public_id,
    user_public_id,
    name,
    token_hash,
    prefix,
    scopes,
    expires_at,
    last_used_at,
    revoked_at,
    created_at
FROM api_tokens
WHERE user_public_id = sqlc.arg('user_public_id')
  AND revoked_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('row_limit') OFFSET sqlc.arg('row_offset');

-- name: CountTokensByUser :one
SELECT count(*)
FROM api_tokens
WHERE user_public_id = sqlc.arg('user_public_id')
  AND revoked_at IS NULL;

-- name: FindTokenByPublicID :one
SELECT
    public_id,
    user_public_id,
    name,
    token_hash,
    prefix,
    scopes,
    expires_at,
    last_used_at,
    revoked_at,
    created_at
FROM api_tokens
WHERE public_id = sqlc.arg('public_id')
LIMIT 1;

-- name: RevokeToken :execrows
UPDATE api_tokens
SET revoked_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND user_public_id = sqlc.arg('user_public_id')
  AND revoked_at IS NULL;

-- name: RevokeTokensByUser :execrows
UPDATE api_tokens
SET revoked_at = now()
WHERE user_public_id = sqlc.arg('user_public_id')
  AND revoked_at IS NULL;

-- name: TouchTokenLastUsed :exec
UPDATE api_tokens
SET last_used_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND revoked_at IS NULL;
