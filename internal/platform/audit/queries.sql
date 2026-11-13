-- name: InsertAuditEvent :one
INSERT INTO audit_events (
    occurred_at, request_id, actor_key, action, resource, method,
    status, outcome, metadata
) VALUES (
    sqlc.arg('occurred_at'),
    sqlc.arg('request_id'),
    sqlc.arg('actor_key'),
    sqlc.arg('action'),
    sqlc.arg('resource'),
    sqlc.arg('method'),
    sqlc.arg('status'),
    sqlc.arg('outcome'),
    sqlc.arg('metadata')::jsonb
)
RETURNING id;

-- name: DeleteAuditBatchBefore :execrows
DELETE FROM audit_events
WHERE id IN (
    SELECT ae.id FROM audit_events ae
    WHERE ae.occurred_at < sqlc.arg('occurred_at')
    ORDER BY ae.occurred_at ASC
    LIMIT sqlc.arg('row_limit')
);

-- name: ListAuditEvents :many
SELECT
    id, occurred_at, request_id, actor_key, action, resource,
    method, status, outcome, metadata::text AS metadata
FROM audit_events
WHERE (
    sqlc.narg('actor_key')::text IS NULL
    OR actor_key = sqlc.narg('actor_key')::text
  )
  AND (
    sqlc.narg('action')::text IS NULL
    OR action = sqlc.narg('action')::text
  )
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg('row_limit')::int;
