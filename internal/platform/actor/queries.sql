-- name: FindActiveActor :one
SELECT 1
FROM users
WHERE public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL
  AND is_active
LIMIT 1;
