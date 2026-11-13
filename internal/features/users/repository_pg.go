package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	usersqlc "kinstack/internal/features/users/sqlc"
	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/pagination"

	"github.com/lib/pq"
)

type SQLRepository struct {
	db      *sql.DB
	queries *usersqlc.Queries
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "users database is required")

	queries, err := usersqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare users queries: %w", err)
	}
	return &SQLRepository{db: dependency.DB, queries: queries}, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

func (r *SQLRepository) ListUsers(
	ctx context.Context,
	query pagination.Query,
) ([]UserResult, error) {
	rows, err := r.queries.ListUsers(ctx, usersqlc.ListUsersParams{
		RowLimit:  int32(query.Limit()),
		RowOffset: int32(query.Offset()),
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	results := make([]UserResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, userFromListRow(row))
	}
	return results, nil
}

func (r *SQLRepository) CountUsers(ctx context.Context) (UserCounts, error) {
	counts, err := r.queries.CountUsers(ctx)
	if err != nil {
		return UserCounts{}, fmt.Errorf("count users: %w", err)
	}
	return UserCounts{Total: int(counts.Total), Active: int(counts.Active)}, nil
}

func (r *SQLRepository) FindUserByPublicID(
	ctx context.Context,
	publicID string,
) (UserResult, error) {
	row, err := r.queries.FindUserByPublicID(ctx, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return UserResult{}, ErrNotFound
	}
	if err != nil {
		return UserResult{}, fmt.Errorf("find user: %w", err)
	}
	return userFromFindRow(row), nil
}

func (r *SQLRepository) CreateUser(ctx context.Context, input CreateUserInput) (UserResult, error) {
	row, err := r.queries.CreateUser(ctx, usersqlc.CreateUserParams{
		PublicID:     input.PublicID,
		Username:     input.Username,
		Email:        input.Email,
		DisplayName:  input.DisplayName,
		PasswordHash: input.PasswordHash,
		Invited:      input.Invited,
	})
	if constraint, ok := uniqueViolationConstraint(err); ok {
		if constraint == "users_email_active_idx" {
			return UserResult{}, ErrEmailTaken
		}
		return UserResult{}, ErrUsernameTaken
	}
	if err != nil {
		return UserResult{}, fmt.Errorf("create user: %w", err)
	}
	return userFromCreateRow(row), nil
}

func (r *SQLRepository) UpdateUser(ctx context.Context, input UpdateUserInput) (UserResult, error) {
	row, err := r.queries.UpdateUser(ctx, usersqlc.UpdateUserParams{
		PublicID:    input.PublicID,
		DisplayName: input.DisplayName,
		Email:       input.Email,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return UserResult{}, ErrNotFound
	}
	if constraint, ok := uniqueViolationConstraint(
		err,
	); ok &&
		constraint == "users_email_active_idx" {
		return UserResult{}, ErrEmailTaken
	}
	if err != nil {
		return UserResult{}, fmt.Errorf("update user: %w", err)
	}
	return userFromUpdateRow(row), nil
}

func (r *SQLRepository) UpdatePassword(ctx context.Context, publicID, hash string) error {
	affected, err := r.queries.UpdateUserPassword(ctx, usersqlc.UpdateUserPasswordParams{
		PublicID:     publicID,
		PasswordHash: hash,
	})
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeactivateGuarded checks and updates in one transaction under an advisory lock, so two
// concurrent deactivations cannot both observe a surviving administrator.
func (r *SQLRepository) DeactivateGuarded(
	ctx context.Context,
	publicID string,
) (UserResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return UserResult{}, fmt.Errorf("begin deactivate transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	queries := r.queries.WithTx(tx)
	if err := queries.LockAccessManagers(ctx); err != nil {
		return UserResult{}, fmt.Errorf("lock access managers: %w", err)
	}

	others, err := queries.CountActiveAccessManagers(ctx, publicID)
	if err != nil {
		return UserResult{}, fmt.Errorf("count access managers: %w", err)
	}
	if others == 0 {
		return UserResult{}, ErrLastAdministrator
	}

	row, err := queries.SetUserActive(ctx, usersqlc.SetUserActiveParams{
		PublicID: publicID,
		IsActive: false,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return UserResult{}, ErrNotFound
	}
	if err != nil {
		return UserResult{}, fmt.Errorf("deactivate account: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return UserResult{}, fmt.Errorf("commit deactivate: %w", err)
	}
	return userFromSetActiveRow(row), nil
}

func (r *SQLRepository) SetActive(
	ctx context.Context,
	publicID string,
	active bool,
) (UserResult, error) {
	row, err := r.queries.SetUserActive(ctx, usersqlc.SetUserActiveParams{
		PublicID: publicID,
		IsActive: active,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return UserResult{}, ErrNotFound
	}
	if err != nil {
		return UserResult{}, fmt.Errorf("set active: %w", err)
	}
	return userFromSetActiveRow(row), nil
}

func userFromListRow(row usersqlc.ListUsersRow) UserResult {
	return UserResult{
		PublicID:    row.PublicID,
		Username:    row.Username,
		Email:       row.Email,
		DisplayName: row.DisplayName,
		Role:        row.Role,
		IsActive:    row.IsActive,
		Invited:     row.Invited,
		LastLoginAt: nullableTime(row.LastLoginAt),
		CreatedAt:   row.CreatedAt,
	}
}

func userFromFindRow(row usersqlc.FindUserByPublicIDRow) UserResult {
	return UserResult{
		PublicID:    row.PublicID,
		Username:    row.Username,
		Email:       row.Email,
		DisplayName: row.DisplayName,
		Role:        row.Role,
		IsActive:    row.IsActive,
		Invited:     row.Invited,
		LastLoginAt: nullableTime(row.LastLoginAt),
		CreatedAt:   row.CreatedAt,
	}
}

func userFromCreateRow(row usersqlc.CreateUserRow) UserResult {
	return UserResult{
		PublicID:    row.PublicID,
		Username:    row.Username,
		Email:       row.Email,
		DisplayName: row.DisplayName,
		Role:        row.Role,
		IsActive:    row.IsActive,
		Invited:     row.Invited,
		LastLoginAt: nullableTime(row.LastLoginAt),
		CreatedAt:   row.CreatedAt,
	}
}

func userFromUpdateRow(row usersqlc.UpdateUserRow) UserResult {
	return UserResult{
		PublicID:    row.PublicID,
		Username:    row.Username,
		Email:       row.Email,
		DisplayName: row.DisplayName,
		Role:        row.Role,
		IsActive:    row.IsActive,
		Invited:     row.Invited,
		LastLoginAt: nullableTime(row.LastLoginAt),
		CreatedAt:   row.CreatedAt,
	}
}

func userFromSetActiveRow(row usersqlc.SetUserActiveRow) UserResult {
	return UserResult{
		PublicID:    row.PublicID,
		Username:    row.Username,
		Email:       row.Email,
		DisplayName: row.DisplayName,
		Role:        row.Role,
		IsActive:    row.IsActive,
		Invited:     row.Invited,
		LastLoginAt: nullableTime(row.LastLoginAt),
		CreatedAt:   row.CreatedAt,
	}
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func uniqueViolationConstraint(err error) (string, bool) {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return pqErr.Constraint, true
	}
	return "", false
}
