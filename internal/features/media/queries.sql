-- name: ListAssets :many
SELECT
    a.public_id,
    u.public_id AS owner_public_id,
    a.filename,
    a.content_type,
    a.size_bytes,
    a.object_key,
    a.created_at
FROM media_assets a
JOIN users u ON u.id = a.owner_id
WHERE u.public_id = sqlc.arg('owner_public_id')
  AND a.deleted_at IS NULL
ORDER BY a.created_at DESC, a.id DESC
LIMIT sqlc.arg('row_limit') OFFSET sqlc.arg('row_offset');

-- name: CountAssets :one
SELECT COUNT(*)
FROM media_assets a
JOIN users u ON u.id = a.owner_id
WHERE u.public_id = sqlc.arg('owner_public_id')
  AND a.deleted_at IS NULL;

-- name: CreateAsset :one
WITH inserted AS (
    INSERT INTO media_assets (public_id, owner_id, filename, content_type, size_bytes, object_key)
    SELECT
        sqlc.arg('public_id'),
        u.id,
        sqlc.arg('filename'),
        sqlc.arg('content_type'),
        sqlc.arg('size_bytes'),
        sqlc.arg('object_key')
    FROM users u
    WHERE u.public_id = sqlc.arg('owner_public_id')
      AND u.deleted_at IS NULL
    RETURNING *
)
SELECT
    a.public_id,
    u.public_id AS owner_public_id,
    a.filename,
    a.content_type,
    a.size_bytes,
    a.object_key,
    a.created_at
FROM inserted a
JOIN users u ON u.id = a.owner_id;

-- name: FindAsset :one
SELECT
    a.public_id,
    u.public_id AS owner_public_id,
    a.filename,
    a.content_type,
    a.size_bytes,
    a.object_key,
    a.created_at
FROM media_assets a
JOIN users u ON u.id = a.owner_id
WHERE u.public_id = sqlc.arg('owner_public_id')
  AND a.public_id = sqlc.arg('public_id')
  AND a.deleted_at IS NULL
LIMIT 1;

-- name: SoftDeleteAsset :one
WITH updated AS (
    UPDATE media_assets a
    SET deleted_at = now()
    WHERE a.owner_id = (
            SELECT u.id FROM users u
            WHERE u.public_id = sqlc.arg('owner_public_id')
        )
      AND a.public_id = sqlc.arg('public_id')
      AND a.deleted_at IS NULL
    RETURNING *
)
SELECT
    a.public_id,
    u.public_id AS owner_public_id,
    a.filename,
    a.content_type,
    a.size_bytes,
    a.object_key,
    a.created_at
FROM updated a
JOIN users u ON u.id = a.owner_id;
