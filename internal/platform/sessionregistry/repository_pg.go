package sessionregistry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"kinstack/internal/pkg/assert"
	platformdatabase "kinstack/internal/platform/database"

	"github.com/jmoiron/sqlx"
)

const sessionColumns = `
s.public_id,
u.public_id AS user_public_id,
s.session_id,
s.ip,
s.user_agent,
s.created_at,
s.last_seen_at,
s.revoked_at`

const (
	registerSessionQuery = `
INSERT INTO user_sessions (public_id, user_id, session_id, ip, user_agent)
SELECT $1, id, $2, $3, $4
FROM users
WHERE public_id = $5
  AND deleted_at IS NULL
ON CONFLICT (session_id) DO UPDATE
SET last_seen_at = now(),
    revoked_at = NULL,
    ip = EXCLUDED.ip,
    user_agent = EXCLUDED.user_agent
RETURNING public_id, $5::text AS user_public_id, session_id, ip, user_agent,
          created_at, last_seen_at, revoked_at`

	touchSessionQuery = `
UPDATE user_sessions
SET last_seen_at = now()
WHERE session_id = $1
  AND revoked_at IS NULL
  AND last_seen_at < now() - interval '60 seconds'`

	listActiveSessionsQuery = `
SELECT ` + sessionColumns + `
FROM user_sessions s
JOIN users u ON u.id = s.user_id
WHERE u.public_id = $1
  AND s.revoked_at IS NULL
ORDER BY s.last_seen_at DESC, s.id DESC`

	findSessionByPublicIDQuery = `
SELECT ` + sessionColumns + `
FROM user_sessions s
JOIN users u ON u.id = s.user_id
WHERE s.public_id = $1
  AND s.revoked_at IS NULL
LIMIT 1`

	revokeSessionQuery = `
UPDATE user_sessions
SET revoked_at = now()
WHERE public_id = $1
  AND revoked_at IS NULL`

	revokeSessionBySessionIDQuery = `
UPDATE user_sessions
SET revoked_at = now()
WHERE session_id = $1
  AND revoked_at IS NULL`

	revokeOtherSessionsQuery = `
UPDATE user_sessions
SET revoked_at = now()
WHERE user_id = (
    SELECT id FROM users WHERE public_id = $1 AND deleted_at IS NULL
  )
  AND session_id <> $2
  AND revoked_at IS NULL
RETURNING session_id`
)

// SQLRepository is the PostgreSQL session registry.
type SQLRepository struct {
	owner      *platformdatabase.StatementOwner
	register   *sqlx.Stmt
	touch      *sqlx.Stmt
	listActive *sqlx.Stmt
	findByID   *sqlx.Stmt
	revoke     *sqlx.Stmt
	revokeByID *sqlx.Stmt
	revokeRest *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

// RepositoryDependency contains the dependencies required to build the repository.
type RepositoryDependency struct {
	DB *sqlx.DB
}

// NewSQLRepository constructs the PostgreSQL session registry.
func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "session registry database is required")

	owner := platformdatabase.NewStatementOwner(dependency.DB)
	repository := &SQLRepository{owner: owner}
	statements := []struct {
		target **sqlx.Stmt
		name   string
		query  string
	}{
		{&repository.register, "session_registry_register", registerSessionQuery},
		{&repository.touch, "session_registry_touch", touchSessionQuery},
		{&repository.listActive, "session_registry_list_active", listActiveSessionsQuery},
		{&repository.findByID, "session_registry_find", findSessionByPublicIDQuery},
		{&repository.revoke, "session_registry_revoke", revokeSessionQuery},
		{&repository.revokeByID, "session_registry_revoke_by_id", revokeSessionBySessionIDQuery},
		{&repository.revokeRest, "session_registry_revoke_rest", revokeOtherSessionsQuery},
	}
	for _, statement := range statements {
		stmt, err := owner.Prepare(statement.name, statement.query)
		if err != nil {
			return nil, err
		}
		*statement.target = stmt
	}
	return repository, nil
}

// Close releases the prepared statements owned by the repository.
func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.owner.Close()
}

// Register records or refreshes one authenticated browser session.
func (r *SQLRepository) Register(
	ctx context.Context,
	sessionPublicID string,
	input RegisterInput,
) (Result, error) {
	var result Result
	if err := r.register.GetContext(
		ctx,
		&result,
		sessionPublicID,
		input.SessionID,
		input.IP,
		input.UserAgent,
		input.UserPublicID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Result{}, ErrNotFound
		}
		return Result{}, fmt.Errorf("register session: %w", err)
	}
	return result, nil
}

// Touch refreshes last_seen_at for a session that has been idle for a while.
func (r *SQLRepository) Touch(ctx context.Context, sessionID string) error {
	if _, err := r.touch.ExecContext(ctx, sessionID); err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

// ListActive returns the active sessions of one user.
func (r *SQLRepository) ListActive(ctx context.Context, userPublicID string) ([]Result, error) {
	results := make([]Result, 0)
	if err := r.listActive.SelectContext(ctx, &results, userPublicID); err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	return results, nil
}

// FindByPublicID returns one active session.
func (r *SQLRepository) FindByPublicID(ctx context.Context, sessionPublicID string) (Result, error) {
	var result Result
	if err := r.findByID.GetContext(ctx, &result, sessionPublicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Result{}, ErrNotFound
		}
		return Result{}, fmt.Errorf("find session: %w", err)
	}
	return result, nil
}

// Revoke marks one session as revoked.
func (r *SQLRepository) Revoke(ctx context.Context, sessionPublicID string) error {
	result, err := r.revoke.ExecContext(ctx, sessionPublicID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke session rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// RevokeBySessionID marks the transport session as revoked on logout.
func (r *SQLRepository) RevokeBySessionID(ctx context.Context, sessionID string) error {
	if _, err := r.revokeByID.ExecContext(ctx, sessionID); err != nil {
		return fmt.Errorf("revoke session by id: %w", err)
	}
	return nil
}

// RevokeAllExcept revokes every active session except the current one and
// returns the transport identifiers to destroy.
func (r *SQLRepository) RevokeAllExcept(
	ctx context.Context,
	userPublicID string,
	keepSessionID string,
) ([]string, error) {
	sessionIDs := make([]string, 0)
	if err := r.revokeRest.SelectContext(ctx, &sessionIDs, userPublicID, keepSessionID); err != nil {
		return nil, fmt.Errorf("revoke other sessions: %w", err)
	}
	return sessionIDs, nil
}
