package twofactor

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

const (
	findStatusQuery = `
SELECT two_factor_enabled, two_factor_secret
FROM users
WHERE public_id = $1
  AND deleted_at IS NULL
LIMIT 1`

	setSecretQuery = `
UPDATE users
SET two_factor_secret = $2,
    two_factor_enabled = false,
    two_factor_confirmed_at = NULL,
    updated_at = now()
WHERE public_id = $1
  AND deleted_at IS NULL`

	enableQuery = `
UPDATE users
SET two_factor_enabled = true,
    two_factor_confirmed_at = now(),
    updated_at = now()
WHERE public_id = $1
  AND deleted_at IS NULL`

	disableQuery = `
UPDATE users
SET two_factor_enabled = false,
    two_factor_secret = '',
    two_factor_confirmed_at = NULL,
    updated_at = now()
WHERE public_id = $1
  AND deleted_at IS NULL`

	deleteRecoveryCodesQuery = `
DELETE FROM user_recovery_codes
WHERE user_id = (
    SELECT id FROM users WHERE public_id = $1 AND deleted_at IS NULL
  )`

	insertRecoveryCodesQuery = `
INSERT INTO user_recovery_codes (user_id, code_hash)
SELECT u.id, h
FROM users u
CROSS JOIN unnest($2::text[]) AS h
WHERE u.public_id = $1
  AND u.deleted_at IS NULL`

	findRecoveryCodeQuery = `
SELECT c.id
FROM user_recovery_codes c
JOIN users u ON u.id = c.user_id
WHERE u.public_id = $1
  AND u.deleted_at IS NULL
  AND c.code_hash = $2
  AND c.used_at IS NULL
LIMIT 1`

	markRecoveryCodeUsedQuery = `
UPDATE user_recovery_codes
SET used_at = now()
WHERE id = $1
  AND used_at IS NULL`
)

// SQLRepository is the PostgreSQL two-factor repository.
type SQLRepository struct {
	owner       *platformdatabase.StatementOwner
	findStatus  *sqlx.Stmt
	setSecret   *sqlx.Stmt
	enable      *sqlx.Stmt
	disable     *sqlx.Stmt
	deleteCodes *sqlx.Stmt
	insertCodes *sqlx.Stmt
	findCode    *sqlx.Stmt
	useCode     *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

// RepositoryDependency contains the dependencies required to build the repository.
type RepositoryDependency struct {
	DB *sqlx.DB
}

// NewSQLRepository constructs the PostgreSQL two-factor repository.
func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "two-factor database is required")

	owner := platformdatabase.NewStatementOwner(dependency.DB)
	repository := &SQLRepository{owner: owner}
	statements := []struct {
		target **sqlx.Stmt
		name   string
		query  string
	}{
		{&repository.findStatus, "two_factor_find_status", findStatusQuery},
		{&repository.setSecret, "two_factor_set_secret", setSecretQuery},
		{&repository.enable, "two_factor_enable", enableQuery},
		{&repository.disable, "two_factor_disable", disableQuery},
		{&repository.deleteCodes, "two_factor_delete_codes", deleteRecoveryCodesQuery},
		{&repository.insertCodes, "two_factor_insert_codes", insertRecoveryCodesQuery},
		{&repository.findCode, "two_factor_find_code", findRecoveryCodeQuery},
		{&repository.useCode, "two_factor_use_code", markRecoveryCodeUsedQuery},
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

// FindStatus returns the two-factor state of one account.
func (r *SQLRepository) FindStatus(ctx context.Context, userPublicID string) (StatusResult, error) {
	var result StatusResult
	if err := r.findStatus.GetContext(ctx, &result, userPublicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return StatusResult{}, ErrNotFound
		}
		return StatusResult{}, fmt.Errorf("find two-factor status: %w", err)
	}
	return result, nil
}

// SetSecret stores a pending secret without enabling two-factor auth.
func (r *SQLRepository) SetSecret(ctx context.Context, userPublicID, secret string) error {
	return r.execAffectingOne(ctx, r.setSecret, "set two-factor secret", userPublicID, secret)
}

// Enable activates two-factor auth for one account.
func (r *SQLRepository) Enable(ctx context.Context, userPublicID string) error {
	return r.execAffectingOne(ctx, r.enable, "enable two-factor", userPublicID)
}

// Disable clears the secret and deactivates two-factor auth.
func (r *SQLRepository) Disable(ctx context.Context, userPublicID string) error {
	return r.execAffectingOne(ctx, r.disable, "disable two-factor", userPublicID)
}

// ReplaceRecoveryCodes deletes the existing codes and stores the new hashes.
func (r *SQLRepository) ReplaceRecoveryCodes(
	ctx context.Context,
	userPublicID string,
	hashes []string,
) error {
	if _, err := r.deleteCodes.ExecContext(ctx, userPublicID); err != nil {
		return fmt.Errorf("delete recovery codes: %w", err)
	}
	if len(hashes) == 0 {
		return nil
	}
	if _, err := r.insertCodes.ExecContext(ctx, userPublicID, pq.Array(hashes)); err != nil {
		return fmt.Errorf("insert recovery codes: %w", err)
	}
	return nil
}

// FindRecoveryCode returns the id of an unused recovery code.
func (r *SQLRepository) FindRecoveryCode(
	ctx context.Context,
	userPublicID string,
	hash string,
) (int64, error) {
	var id int64
	if err := r.findCode.GetContext(ctx, &id, userPublicID, hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("find recovery code: %w", err)
	}
	return id, nil
}

// MarkRecoveryCodeUsed consumes a recovery code so it cannot be replayed.
func (r *SQLRepository) MarkRecoveryCodeUsed(ctx context.Context, id int64) error {
	return r.execAffectingOne(ctx, r.useCode, "mark recovery code used", id)
}

func (r *SQLRepository) execAffectingOne(
	ctx context.Context,
	stmt *sqlx.Stmt,
	action string,
	args ...any,
) error {
	result, err := stmt.ExecContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s rows: %w", action, err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
