-- name: ListNotifications :many
SELECT
    id,
    public_id,
    recipient_public_id,
    kind,
    title,
    body,
    status,
    created_at,
    read_at
FROM notifications
WHERE recipient_public_id = sqlc.arg('recipient_public_id')
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('row_limit');

-- name: ListNotificationsBefore :many
SELECT
    id,
    public_id,
    recipient_public_id,
    kind,
    title,
    body,
    status,
    created_at,
    read_at
FROM notifications
WHERE recipient_public_id = sqlc.arg('recipient_public_id')
  AND (created_at, id) < (sqlc.arg('before')::timestamptz, sqlc.arg('before_id')::bigint)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('row_limit')::int;

-- name: CountUnreadNotifications :one
SELECT count(*)
FROM notifications
WHERE recipient_public_id = sqlc.arg('recipient_public_id')
  AND status = 'unread';

-- name: CreateNotification :one
INSERT INTO notifications (public_id, recipient_public_id, kind, title, body)
VALUES (
    sqlc.arg('public_id'),
    sqlc.arg('recipient_public_id'),
    sqlc.arg('kind'),
    sqlc.arg('title'),
    sqlc.arg('body')
)
RETURNING
    id,
    public_id,
    recipient_public_id,
    kind,
    title,
    body,
    status,
    created_at,
    read_at;

-- name: MarkNotificationRead :one
UPDATE notifications
SET status = 'read',
    read_at = COALESCE(read_at, now())
WHERE recipient_public_id = sqlc.arg('recipient_public_id')
  AND public_id = sqlc.arg('public_id')
RETURNING
    id,
    public_id,
    recipient_public_id,
    kind,
    title,
    body,
    status,
    created_at,
    read_at;

-- name: MarkAllNotificationsRead :execrows
UPDATE notifications
SET status = 'read',
    read_at = COALESCE(read_at, now())
WHERE recipient_public_id = sqlc.arg('recipient_public_id')
  AND status = 'unread';
