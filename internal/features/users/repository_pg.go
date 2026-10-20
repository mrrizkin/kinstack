package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"kinstack/internal/pkg/assert"
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
	listUsersQuery = `
SELECT ` + userProjection + `
FROM users u
WHERE u.deleted_at IS NULL
ORDER BY u.created_at ASC, u.id ASC`

	findUserByPublicIDQuery = `
SELECT ` + userProjection + `
FROM users u
WHERE u.public_id = $1
  AND u.deleted_at IS NULL
LIMIT 1`

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

// SQLRepository is the PostgreSQL user account repository.
type SQLRepository struct {
	owner        *platformdatabase.StatementOwner
	listStmt     *sqlx.Stmt
	findStmt     *sqlx.Stmt
	createStmt   *sqlx.Stmt
	updateStmt   *sqlx.Stmt
	passwordStmt *sqlx.Stmt
	activeStmt   *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

// RepositoryDependency contains the dependencies required to construct the SQL repository.
type RepositoryDependency struct {
	DB *sqlx.DB
}

// NewSQLRepository constructs the PostgreSQL user repository.
func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "users database is required")

	owner := platformdatabase.NewStatementOwner(dependency.DB)
	repository := &SQLRepository{owner: owner}
	statements := []struct {
		target **sqlx.Stmt
		name   string
		query  string
	}{
		{&repository.listStmt, "users_list", listUsersQuery},
		{&repository.findStmt, "users_find_by_public_id", findUserByPublicIDQuery},
		{&repository.createStmt, "users_create", createUserQuery},
		{&repository.updateStmt, "users_update", updateUserQuery},
		{&repository.passwordStmt, "users_update_password", updatePasswordQuery},
		{&repository.activeStmt, "users_set_active", setActiveQuery},
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

// ListUsers returns every non-deleted account.
func (r *SQLRepository) ListUsers(ctx context.Context) ([]UserResult, error) {
	rows := make([]UserResult, 0)
	if err := r.listStmt.SelectContext(ctx, &rows); err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return rows, nil
}

// FindUserByPublicID resolves a single account.
func (r *SQLRepository) FindUserByPublicID(
	ctx context.Context,
	publicID string,
) (UserResult, error) {
	var result UserResult
	err := r.findStmt.GetContext(ctx, &result, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return UserResult{}, ErrNotFound
	}
	if err != nil {
		return UserResult{}, fmt.Errorf("find user: %w", err)
	}
	return result, nil
}

// CreateUser inserts a new account and maps duplicate usernames to ErrUsernameTaken.
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

// UpdateUser edits an account display name.
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
	if constraint, ok := uniqueViolationConstraint(err); ok && constraint == "users_email_active_idx" {
		return UserResult{}, ErrEmailTaken
	}
	if err != nil {
		return UserResult{}, fmt.Errorf("update user: %w", err)
	}
	return result, nil
}

// UpdatePassword replaces an account password hash.
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

// SetActive enables or disables an account.
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
