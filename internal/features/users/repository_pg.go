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
	platformdatabase "kinstack/internal/platform/database"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const userProjection = `
u.public_id,
u.username,
u.email,
u.display_name,
COALESCE((
    SELECT ur.role_code
    FROM user_roles ur
    JOIN roles r ON r.code = ur.role_code
    WHERE ur.user_public_id = u.public_id
    ORDER BY r.rank DESC, ur.role_code ASC
    LIMIT 1
), '') AS role,
u.is_active,
(u.invited_at IS NOT NULL) AS invited,
u.last_login_at,
u.created_at`

const (
	createUserQuery = `
WITH inserted AS (
    INSERT INTO users (public_id, username, email, display_name, password_hash, invited_at)
    VALUES ($1, $2, $3, $4, $5, CASE WHEN $6 THEN now() ELSE NULL END)
    RETURNING *
)
SELECT ` + userProjection + `
FROM inserted u`

	updateUserQuery = `
WITH updated AS (
    UPDATE users
    SET display_name = $2, email = $3, updated_at = now()
    WHERE public_id = $1
      AND deleted_at IS NULL
    RETURNING *
)
SELECT ` + userProjection + `
FROM updated u`

	updatePasswordQuery = `
UPDATE users
SET password_hash = $2, updated_at = now()
WHERE public_id = $1
  AND deleted_at IS NULL`

	// lockAccessManagers serializes every decision that could remove the last
	// account able to manage access. The key is a constant, so all replicas
	// contend on the same advisory lock.
	lockAccessManagersQuery = `SELECT pg_advisory_xact_lock(hashtext('kinstack:access_managers'))`

	countActiveAccessManagersQuery = `
SELECT COUNT(*)
FROM users u
WHERE u.public_id <> $1
  AND u.is_active
  AND u.deleted_at IS NULL
  AND (
    EXISTS (
        SELECT 1
        FROM user_roles ur
        JOIN role_permissions rp ON rp.role_code = ur.role_code
        WHERE ur.user_public_id = u.public_id
          AND rp.permission_code = 'access.manage'
      )
    OR EXISTS (
        SELECT 1
        FROM user_permissions up
        WHERE up.user_public_id = u.public_id
          AND up.permission_code = 'access.manage'
          AND up.effect = 'allow'
          AND (up.expires_at IS NULL OR up.expires_at > now())
      )
  )
  AND NOT EXISTS (
    SELECT 1
    FROM user_permissions up
    WHERE up.user_public_id = u.public_id
      AND up.permission_code = 'access.manage'
      AND up.effect = 'deny'
      AND (up.expires_at IS NULL OR up.expires_at > now())
  )`

	setActiveQuery = `
WITH updated AS (
    UPDATE users
    SET is_active = $2, updated_at = now()
    WHERE public_id = $1
      AND deleted_at IS NULL
    RETURNING *
)
SELECT ` + userProjection + `
FROM updated u`
)

type SQLRepository struct {
	db    *sqlx.DB
	owner *platformdatabase.StatementOwner
	// reads are generated from queries.sql; writes stay prepared here because
	// they carry error classification and RETURNING projections.
	reads        *usersqlc.Queries
	createStmt   *sqlx.Stmt
	updateStmt   *sqlx.Stmt
	passwordStmt *sqlx.Stmt
	activeStmt   *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sqlx.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "users database is required")

	reads, err := usersqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare users reads: %w", err)
	}

	owner := platformdatabase.NewStatementOwner(dependency.DB)
	repository := &SQLRepository{db: dependency.DB, owner: owner, reads: reads}
	statements := []struct {
		target **sqlx.Stmt
		name   string
		query  string
	}{
		{&repository.createStmt, "users_create", createUserQuery},
		{&repository.updateStmt, "users_update", updateUserQuery},
		{&repository.passwordStmt, "users_update_password", updatePasswordQuery},
		{&repository.activeStmt, "users_set_active", setActiveQuery},
	}
	for _, statement := range statements {
		stmt, err := owner.Prepare(statement.name, statement.query)
		if err != nil {
			return nil, errors.Join(err, reads.Close())
		}
		*statement.target = stmt
	}
	return repository, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return errors.Join(r.reads.Close(), r.owner.Close())
}

func (r *SQLRepository) ListUsers(
	ctx context.Context,
	query pagination.Query,
) ([]UserResult, error) {
	rows, err := r.reads.ListUsers(ctx, usersqlc.ListUsersParams{
		Limit:  int32(query.Limit()),
		Offset: int32(query.Offset()),
	})
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	results := make([]UserResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, UserResult{
			PublicID:    row.PublicID,
			Username:    row.Username,
			Email:       row.Email,
			DisplayName: row.DisplayName,
			Role:        row.Role,
			IsActive:    row.IsActive,
			Invited:     row.Invited,
			LastLoginAt: nullableTime(row.LastLoginAt),
			CreatedAt:   row.CreatedAt,
		})
	}
	return results, nil
}

func (r *SQLRepository) CountUsers(ctx context.Context) (UserCounts, error) {
	counts, err := r.reads.CountUsers(ctx)
	if err != nil {
		return UserCounts{}, fmt.Errorf("count users: %w", err)
	}
	return UserCounts{Total: int(counts.Total), Active: int(counts.Active)}, nil
}

func (r *SQLRepository) FindUserByPublicID(
	ctx context.Context,
	publicID string,
) (UserResult, error) {
	row, err := r.reads.FindUserByPublicID(ctx, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return UserResult{}, ErrNotFound
	}
	if err != nil {
		return UserResult{}, fmt.Errorf("find user: %w", err)
	}
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
	}, nil
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func (r *SQLRepository) CreateUser(ctx context.Context, input CreateUserInput) (UserResult, error) {
	var result UserResult
	err := r.createStmt.GetContext(
		ctx,
		&result,
		input.PublicID,
		input.Username,
		input.Email,
		input.DisplayName,
		input.PasswordHash,
		input.Invited,
	)
	if constraint, ok := uniqueViolationConstraint(err); ok {
		if constraint == "users_email_active_idx" {
			return UserResult{}, ErrEmailTaken
		}
		return UserResult{}, ErrUsernameTaken
	}
	if err != nil {
		return UserResult{}, fmt.Errorf("create user: %w", err)
	}
	return result, nil
}

func (r *SQLRepository) UpdateUser(ctx context.Context, input UpdateUserInput) (UserResult, error) {
	var result UserResult
	err := r.updateStmt.GetContext(
		ctx,
		&result,
		input.PublicID,
		input.DisplayName,
		input.Email,
	)
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
	return result, nil
}

func (r *SQLRepository) UpdatePassword(ctx context.Context, publicID, hash string) error {
	result, err := r.passwordStmt.ExecContext(ctx, publicID, hash)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update password rows affected: %w", err)
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
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return UserResult{}, fmt.Errorf("begin deactivate transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, lockAccessManagersQuery); err != nil {
		return UserResult{}, fmt.Errorf("lock access managers: %w", err)
	}

	var others int
	if err := tx.GetContext(ctx, &others, countActiveAccessManagersQuery, publicID); err != nil {
		return UserResult{}, fmt.Errorf("count access managers: %w", err)
	}
	if others == 0 {
		return UserResult{}, ErrLastAdministrator
	}

	var result UserResult
	err = tx.GetContext(ctx, &result, setActiveQuery, publicID, false)
	if errors.Is(err, sql.ErrNoRows) {
		return UserResult{}, ErrNotFound
	}
	if err != nil {
		return UserResult{}, fmt.Errorf("deactivate account: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return UserResult{}, fmt.Errorf("commit deactivate: %w", err)
	}
	return result, nil
}

func (r *SQLRepository) SetActive(
	ctx context.Context,
	publicID string,
	active bool,
) (UserResult, error) {
	var result UserResult
	err := r.activeStmt.GetContext(ctx, &result, publicID, active)
	if errors.Is(err, sql.ErrNoRows) {
		return UserResult{}, ErrNotFound
	}
	if err != nil {
		return UserResult{}, fmt.Errorf("set active: %w", err)
	}
	return result, nil
}

func uniqueViolationConstraint(err error) (string, bool) {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return pqErr.Constraint, true
	}
	return "", false
}
