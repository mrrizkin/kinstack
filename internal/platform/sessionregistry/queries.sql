-- name: RegisterSession :one
INSERT INTO user_sessions (public_id, user_id, session_id, ip, user_agent)
SELECT
    sqlc.arg('public_id'),
    u.id,
    sqlc.arg('session_id'),
    sqlc.arg('ip'),
    sqlc.arg('user_agent')
FROM users u
WHERE u.public_id = sqlc.arg('user_public_id')
  AND u.deleted_at IS NULL
ON CONFLICT (session_id) DO UPDATE
SET last_seen_at = now(),
    revoked_at = NULL,
    ip = EXCLUDED.ip,
    user_agent = EXCLUDED.user_agent
RETURNING
    public_id,
    sqlc.arg('user_public_id')::text AS user_public_id,
    session_id,
    ip,
    user_agent,
    created_at,
    last_seen_at,
    revoked_at;

-- name: TouchSession :exec
UPDATE user_sessions
SET last_seen_at = now()
WHERE session_id = sqlc.arg('session_id')
  AND revoked_at IS NULL
  AND last_seen_at < now() - interval '60 seconds';

-- name: ListActiveSessions :many
SELECT
    s.public_id,
    u.public_id AS user_public_id,
    s.session_id,
    s.ip,
    s.user_agent,
    s.created_at,
    s.last_seen_at,
    s.revoked_at
FROM user_sessions s
JOIN users u ON u.id = s.user_id
WHERE u.public_id = sqlc.arg('user_public_id')
  AND s.revoked_at IS NULL
ORDER BY s.last_seen_at DESC, s.id DESC
LIMIT 50;

-- name: FindSessionByPublicID :one
SELECT
    s.public_id,
    u.public_id AS user_public_id,
    s.session_id,
    s.ip,
    s.user_agent,
    s.created_at,
    s.last_seen_at,
    s.revoked_at
FROM user_sessions s
JOIN users u ON u.id = s.user_id
WHERE s.public_id = sqlc.arg('public_id')
  AND s.revoked_at IS NULL
LIMIT 1;

-- name: RevokeSession :execrows
UPDATE user_sessions
SET revoked_at = now()
WHERE public_id = sqlc.arg('public_id')
  AND revoked_at IS NULL;

-- name: RevokeSessionBySessionID :exec
UPDATE user_sessions
SET revoked_at = now()
WHERE session_id = sqlc.arg('session_id')
  AND revoked_at IS NULL;

-- name: RevokeOtherSessions :many
UPDATE user_sessions
SET revoked_at = now()
WHERE user_id = (
    SELECT u.id FROM users u
    WHERE u.public_id = sqlc.arg('user_public_id')
      AND u.deleted_at IS NULL
  )
  AND session_id <> sqlc.arg('keep_session_id')
  AND revoked_at IS NULL
RETURNING session_id;

-- name: RevokeAllSessions :many
UPDATE user_sessions
SET revoked_at = now()
WHERE user_id = (
    SELECT u.id FROM users u
    WHERE u.public_id = sqlc.arg('user_public_id')
      AND u.deleted_at IS NULL
  )
  AND revoked_at IS NULL
RETURNING session_id;
