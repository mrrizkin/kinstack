-- name: ListAuditEvents :many
SELECT
    id,
    occurred_at,
    request_id,
    actor_key,
    action,
    resource,
    method,
    status,
    outcome,
    metadata::text AS metadata
FROM audit_events
WHERE (sqlc.narg('actor_key')::text IS NULL OR actor_key = sqlc.narg('actor_key')::text)
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action')::text)
  AND (sqlc.narg('outcome')::text IS NULL OR outcome = sqlc.narg('outcome')::text)
  AND (
    sqlc.narg('search')::text IS NULL
    OR resource ILIKE sqlc.narg('search')::text
    OR request_id ILIKE sqlc.narg('search')::text
    OR actor_key ILIKE sqlc.narg('search')::text
    OR action ILIKE sqlc.narg('search')::text
  )
  AND (sqlc.narg('from_time')::timestamptz IS NULL OR occurred_at >= sqlc.narg('from_time')::timestamptz)
  AND (sqlc.narg('to_time')::timestamptz IS NULL OR occurred_at < sqlc.narg('to_time')::timestamptz)
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg('row_limit')::int OFFSET sqlc.arg('row_offset')::int;

-- name: CountAuditEvents :one
SELECT count(*)
FROM audit_events
WHERE (sqlc.narg('actor_key')::text IS NULL OR actor_key = sqlc.narg('actor_key')::text)
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action')::text)
  AND (sqlc.narg('outcome')::text IS NULL OR outcome = sqlc.narg('outcome')::text)
  AND (
    sqlc.narg('search')::text IS NULL
    OR resource ILIKE sqlc.narg('search')::text
    OR request_id ILIKE sqlc.narg('search')::text
    OR actor_key ILIKE sqlc.narg('search')::text
    OR action ILIKE sqlc.narg('search')::text
  )
  AND (sqlc.narg('from_time')::timestamptz IS NULL OR occurred_at >= sqlc.narg('from_time')::timestamptz)
  AND (sqlc.narg('to_time')::timestamptz IS NULL OR occurred_at < sqlc.narg('to_time')::timestamptz);
