package sessionregistry

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"kinstack/internal/pkg/assert"
	sessionsqlc "kinstack/internal/platform/sessionregistry/sqlc"
)

type SQLRepository struct {
	queries *sessionsqlc.Queries
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "session registry database is required")

	queries, err := sessionsqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare session registry queries: %w", err)
	}
	return &SQLRepository{queries: queries}, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

func (r *SQLRepository) Register(
	ctx context.Context,
	sessionPublicID string,
	input RegisterInput,
) (Result, error) {
	row, err := r.queries.RegisterSession(ctx, sessionsqlc.RegisterSessionParams{
		PublicID:     sessionPublicID,
		SessionID:    input.SessionID,
		Ip:           input.IP,
		UserAgent:    input.UserAgent,
		UserPublicID: input.UserPublicID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, ErrNotFound
	}
	if err != nil {
		return Result{}, fmt.Errorf("register session: %w", err)
	}
	return Result{
		PublicID:     row.PublicID,
		UserPublicID: row.UserPublicID,
		SessionID:    row.SessionID,
		IP:           row.Ip,
		UserAgent:    row.UserAgent,
		CreatedAt:    row.CreatedAt,
		LastSeenAt:   row.LastSeenAt,
		RevokedAt:    nullableTime(row.RevokedAt),
	}, nil
}

func (r *SQLRepository) Touch(ctx context.Context, sessionID string) error {
	if err := r.queries.TouchSession(ctx, sessionID); err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

func (r *SQLRepository) ListActive(ctx context.Context, userPublicID string) ([]Result, error) {
	rows, err := r.queries.ListActiveSessions(ctx, userPublicID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	results := make([]Result, 0, len(rows))
	for _, row := range rows {
		results = append(results, Result{
			PublicID:     row.PublicID,
			UserPublicID: row.UserPublicID,
			SessionID:    row.SessionID,
			IP:           row.Ip,
			UserAgent:    row.UserAgent,
			CreatedAt:    row.CreatedAt,
			LastSeenAt:   row.LastSeenAt,
			RevokedAt:    nullableTime(row.RevokedAt),
		})
	}
	return results, nil
}

func (r *SQLRepository) FindByPublicID(
	ctx context.Context,
	sessionPublicID string,
) (Result, error) {
	row, err := r.queries.FindSessionByPublicID(ctx, sessionPublicID)
	if errors.Is(err, sql.ErrNoRows) {
		return Result{}, ErrNotFound
	}
	if err != nil {
		return Result{}, fmt.Errorf("find session: %w", err)
	}
	return Result{
		PublicID:     row.PublicID,
		UserPublicID: row.UserPublicID,
		SessionID:    row.SessionID,
		IP:           row.Ip,
		UserAgent:    row.UserAgent,
		CreatedAt:    row.CreatedAt,
		LastSeenAt:   row.LastSeenAt,
		RevokedAt:    nullableTime(row.RevokedAt),
	}, nil
}

func (r *SQLRepository) Revoke(ctx context.Context, sessionPublicID string) error {
	affected, err := r.queries.RevokeSession(ctx, sessionPublicID)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *SQLRepository) RevokeBySessionID(ctx context.Context, sessionID string) error {
	if err := r.queries.RevokeSessionBySessionID(ctx, sessionID); err != nil {
		return fmt.Errorf("revoke session by id: %w", err)
	}
	return nil
}

// RevokeAll revokes every active session of one user and returns the transport
// identifiers to destroy.
func (r *SQLRepository) RevokeAll(ctx context.Context, userPublicID string) ([]string, error) {
	sessionIDs, err := r.queries.RevokeAllSessions(ctx, userPublicID)
	if err != nil {
		return nil, fmt.Errorf("revoke all sessions: %w", err)
	}
	return sessionIDs, nil
}

// RevokeAllExcept revokes every active session except the current one and
// returns the transport identifiers to destroy.
func (r *SQLRepository) RevokeAllExcept(
	ctx context.Context,
	userPublicID string,
	keepSessionID string,
) ([]string, error) {
	sessionIDs, err := r.queries.RevokeOtherSessions(ctx, sessionsqlc.RevokeOtherSessionsParams{
		UserPublicID:  userPublicID,
		KeepSessionID: keepSessionID,
	})
	if err != nil {
		return nil, fmt.Errorf("revoke other sessions: %w", err)
	}
	return sessionIDs, nil
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
