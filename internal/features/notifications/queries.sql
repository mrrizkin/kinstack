-- name: ListNotifications :many
SELECT
    n.id,
    n.public_id,
    u.public_id AS recipient_public_id,
    n.kind,
    n.title,
    n.body,
    n.status,
    n.created_at,
    n.read_at
FROM notifications n
JOIN users u ON u.id = n.recipient_id
WHERE u.public_id = sqlc.arg('recipient_public_id')
ORDER BY n.created_at DESC, n.id DESC
LIMIT sqlc.arg('row_limit');

-- name: ListNotificationsBefore :many
SELECT
    n.id,
    n.public_id,
    u.public_id AS recipient_public_id,
    n.kind,
    n.title,
    n.body,
    n.status,
    n.created_at,
    n.read_at
FROM notifications n
JOIN users u ON u.id = n.recipient_id
WHERE u.public_id = sqlc.arg('recipient_public_id')
  AND (n.created_at, n.id) < (sqlc.arg('before')::timestamptz, sqlc.arg('before_id')::bigint)
ORDER BY n.created_at DESC, n.id DESC
LIMIT sqlc.arg('row_limit')::int;

-- name: CountUnreadNotifications :one
SELECT count(*)
FROM notifications n
JOIN users u ON u.id = n.recipient_id
WHERE u.public_id = sqlc.arg('recipient_public_id')
  AND n.status = 'unread';

-- name: CreateNotification :one
WITH inserted AS (
    INSERT INTO notifications (public_id, recipient_id, kind, title, body)
    SELECT
        sqlc.arg('public_id'),
        u.id,
        sqlc.arg('kind'),
        sqlc.arg('title'),
        sqlc.arg('body')
    FROM users u
    WHERE u.public_id = sqlc.arg('recipient_public_id')
      AND u.deleted_at IS NULL
    RETURNING *
)
SELECT
    n.id,
    n.public_id,
    u.public_id AS recipient_public_id,
    n.kind,
    n.title,
    n.body,
    n.status,
    n.created_at,
    n.read_at
FROM inserted n
JOIN users u ON u.id = n.recipient_id;

-- name: MarkNotificationRead :one
WITH updated AS (
    UPDATE notifications n
    SET status = 'read',
        read_at = COALESCE(read_at, now())
    WHERE n.recipient_id = (
            SELECT u.id FROM users u
            WHERE u.public_id = sqlc.arg('recipient_public_id')
        )
      AND n.public_id = sqlc.arg('public_id')
    RETURNING *
)
SELECT
    n.id,
    n.public_id,
    u.public_id AS recipient_public_id,
    n.kind,
    n.title,
    n.body,
    n.status,
    n.created_at,
    n.read_at
FROM updated n
JOIN users u ON u.id = n.recipient_id;

-- name: MarkAllNotificationsRead :execrows
UPDATE notifications
SET status = 'read',
    read_at = COALESCE(read_at, now())
WHERE recipient_id = (
    SELECT u.id FROM users u
    WHERE u.public_id = sqlc.arg('recipient_public_id')
)
  AND status = 'unread';
