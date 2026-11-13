-- name: ListAssets :many
SELECT
    public_id,
    owner_public_id,
    filename,
    content_type,
    size_bytes,
    object_key,
    created_at
FROM media_assets
WHERE owner_public_id = sqlc.arg('owner_public_id')
  AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('row_limit') OFFSET sqlc.arg('row_offset');

-- name: CountAssets :one
SELECT COUNT(*)
FROM media_assets
WHERE owner_public_id = sqlc.arg('owner_public_id')
  AND deleted_at IS NULL;

-- name: CreateAsset :one
INSERT INTO media_assets (public_id, owner_public_id, filename, content_type, size_bytes, object_key)
VALUES (
    sqlc.arg('public_id'),
    sqlc.arg('owner_public_id'),
    sqlc.arg('filename'),
    sqlc.arg('content_type'),
    sqlc.arg('size_bytes'),
    sqlc.arg('object_key')
)
RETURNING
    public_id,
    owner_public_id,
    filename,
    content_type,
    size_bytes,
    object_key,
    created_at;

-- name: FindAsset :one
SELECT
    public_id,
    owner_public_id,
    filename,
    content_type,
    size_bytes,
    object_key,
    created_at
FROM media_assets
WHERE owner_public_id = sqlc.arg('owner_public_id')
  AND public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL
LIMIT 1;

-- name: SoftDeleteAsset :one
UPDATE media_assets
SET deleted_at = now()
WHERE owner_public_id = sqlc.arg('owner_public_id')
  AND public_id = sqlc.arg('public_id')
  AND deleted_at IS NULL
RETURNING
    public_id,
    owner_public_id,
    filename,
    content_type,
    size_bytes,
    object_key,
    created_at;
