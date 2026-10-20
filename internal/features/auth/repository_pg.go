package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"kinstack/internal/pkg/assert"
	platformdatabase "kinstack/internal/platform/database"

	"github.com/jmoiron/sqlx"
)

const (
	findUserByUsernameQuery = `
SELECT u.public_id, u.username, u.email, u.display_name, u.password_hash,
       COALESCE((
           SELECT ur.role_code
           FROM user_roles ur
           JOIN roles r ON r.code = ur.role_code
           WHERE ur.user_public_id = u.public_id
           ORDER BY r.rank DESC, ur.role_code ASC
           LIMIT 1
       ), '') AS role,
       u.is_active
FROM users u
WHERE lower(u.username) = lower($1)
  AND u.deleted_at IS NULL
LIMIT 1`

	findUserByPublicIDQuery = `
SELECT u.public_id, u.username, u.email, u.display_name, u.password_hash,
       COALESCE((
           SELECT ur.role_code
           FROM user_roles ur
           JOIN roles r ON r.code = ur.role_code
           WHERE ur.user_public_id = u.public_id
           ORDER BY r.rank DESC, ur.role_code ASC
           LIMIT 1
       ), '') AS role,
       u.is_active
FROM users u
WHERE u.public_id = $1
  AND u.deleted_at IS NULL
LIMIT 1`

	findUserByEmailQuery = `
SELECT u.public_id, u.username, u.email, u.display_name, u.password_hash,
       COALESCE((
           SELECT ur.role_code
           FROM user_roles ur
           JOIN roles r ON r.code = ur.role_code
           WHERE ur.user_public_id = u.public_id
           ORDER BY r.rank DESC, ur.role_code ASC
           LIMIT 1
       ), '') AS role,
       u.is_active
FROM users u
WHERE lower(u.email) = lower($1)
  AND u.email <> ''
  AND u.deleted_at IS NULL
LIMIT 1`

	findIdentityQuery = `
SELECT u.public_id, u.username, u.email, u.display_name, COALESCE(u.password_hash, '') AS password_hash,
       COALESCE((
           SELECT ur.role_code
           FROM user_roles ur
           JOIN roles r ON r.code = ur.role_code
           WHERE ur.user_public_id = u.public_id
           ORDER BY r.rank DESC, ur.role_code ASC
           LIMIT 1
       ), '') AS role,
       u.is_active
FROM user_identities i
JOIN users u ON u.id = i.user_id
WHERE i.provider = $1
  AND i.subject = $2
  AND u.deleted_at IS NULL
LIMIT 1`

	upsertIdentityQuery = `
INSERT INTO user_identities (user_id, provider, subject, email)
SELECT id, $2, $3, $4 FROM users WHERE public_id = $1 AND deleted_at IS NULL
ON CONFLICT (provider, subject)
DO UPDATE SET email = EXCLUDED.email, updated_at = now()
WHERE user_identities.user_id = EXCLUDED.user_id`

	touchLastLoginQuery = `
UPDATE users
SET last_login_at = now()
WHERE public_id = $1
  AND deleted_at IS NULL`

	createMagicLinkTokenQuery = `
INSERT INTO magic_link_tokens (user_public_id, token_hash, expires_at)
VALUES ($1, $2, $3)`

	consumeMagicLinkTokenQuery = `
UPDATE magic_link_tokens
SET used_at = now()
WHERE token_hash = $1
  AND used_at IS NULL
  AND expires_at > now()
RETURNING user_public_id`

	createPasswordResetTokenQuery = `
INSERT INTO password_reset_tokens (user_public_id, token_hash, expires_at)
VALUES ($1, $2, $3)`

	consumePasswordResetTokenQuery = `
UPDATE password_reset_tokens
SET used_at = now()
WHERE token_hash = $1
  AND used_at IS NULL
  AND expires_at > now()
RETURNING user_public_id`

	updatePasswordHashQuery = `
UPDATE users
SET password_hash = $2,
    invited_at = NULL,
    updated_at = now()
WHERE public_id = $1
  AND deleted_at IS NULL`
)

// SQLRepository is the PostgreSQL credential repository. Prepared statements
// are owned for the repository lifetime.
type SQLRepository struct {
	owner              *platformdatabase.StatementOwner
	findStmt           *sqlx.Stmt
	findIDStmt         *sqlx.Stmt
	findEmailStmt      *sqlx.Stmt
	findIdentityStmt   *sqlx.Stmt
	upsertIdentityStmt *sqlx.Stmt
	touchStmt          *sqlx.Stmt
	createTokenStmt    *sqlx.Stmt
	consumeTokenStmt   *sqlx.Stmt
	createResetStmt    *sqlx.Stmt
	consumeResetStmt   *sqlx.Stmt
	updatePassStmt     *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

// RepositoryDependency contains the dependencies required to construct the SQL repository.
type RepositoryDependency struct {
	DB *sqlx.DB
}

// NewSQLRepository constructs the PostgreSQL credential repository.
func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "auth database is required")

	owner := platformdatabase.NewStatementOwner(dependency.DB)
	repository := &SQLRepository{owner: owner}
	statements := []struct {
		target **sqlx.Stmt
		name   string
		query  string
	}{
		{&repository.findStmt, "auth_find_user_by_username", findUserByUsernameQuery},
		{&repository.findIDStmt, "auth_find_user_by_public_id", findUserByPublicIDQuery},
		{&repository.findEmailStmt, "auth_find_user_by_email", findUserByEmailQuery},
		{&repository.findIdentityStmt, "auth_find_identity_user", findIdentityQuery},
		{&repository.upsertIdentityStmt, "auth_upsert_identity", upsertIdentityQuery},
		{&repository.touchStmt, "auth_touch_last_login", touchLastLoginQuery},
		{&repository.createTokenStmt, "auth_create_magic_link_token", createMagicLinkTokenQuery},
		{&repository.consumeTokenStmt, "auth_consume_magic_link_token", consumeMagicLinkTokenQuery},
		{&repository.createResetStmt, "auth_create_password_reset_token", createPasswordResetTokenQuery},
		{&repository.consumeResetStmt, "auth_consume_password_reset_token", consumePasswordResetTokenQuery},
		{&repository.updatePassStmt, "auth_update_password_hash", updatePasswordHashQuery},
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

// FindUserByUsername resolves an active account by its login name.
func (r *SQLRepository) FindUserByUsername(
	ctx context.Context,
	username string,
) (FindUserByUsernameResult, error) {
	var result FindUserByUsernameResult
	err := r.findStmt.GetContext(ctx, &result, username)
	if errors.Is(err, sql.ErrNoRows) {
		return FindUserByUsernameResult{}, ErrNotFound
	}
	if err != nil {
		return FindUserByUsernameResult{}, fmt.Errorf("find user by username: %w", err)
	}
	return result, nil
}

// FindUserByPublicID resolves an active account by its public identifier.
func (r *SQLRepository) FindUserByPublicID(
	ctx context.Context,
	publicID string,
) (FindUserByUsernameResult, error) {
	var result FindUserByUsernameResult
	err := r.findIDStmt.GetContext(ctx, &result, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return FindUserByUsernameResult{}, ErrNotFound
	}
	if err != nil {
		return FindUserByUsernameResult{}, fmt.Errorf("find user by public ID: %w", err)
	}
	return result, nil
}

// FindUserByEmail resolves an active account by its email address.
func (r *SQLRepository) FindUserByEmail(
	ctx context.Context,
	email string,
) (FindUserByUsernameResult, error) {
	var result FindUserByUsernameResult
	err := r.findEmailStmt.GetContext(ctx, &result, email)
	if errors.Is(err, sql.ErrNoRows) {
		return FindUserByUsernameResult{}, ErrNotFound
	}
	if err != nil {
		return FindUserByUsernameResult{}, fmt.Errorf("find user by email: %w", err)
	}
	return result, nil
}

// FindIdentityUser resolves an account by its external provider subject.
func (r *SQLRepository) FindIdentityUser(
	ctx context.Context,
	provider string,
	subject string,
) (FindUserByUsernameResult, error) {
	var result FindUserByUsernameResult
	err := r.findIdentityStmt.GetContext(ctx, &result, provider, subject)
	if errors.Is(err, sql.ErrNoRows) {
		return FindUserByUsernameResult{}, ErrNotFound
	}
	if err != nil {
		return FindUserByUsernameResult{}, fmt.Errorf("find identity user: %w", err)
	}
	return result, nil
}

// UpsertIdentity binds an external subject to a local account. The unique key
// is (provider, subject), so the same subject always resolves to one account.
func (r *SQLRepository) UpsertIdentity(
	ctx context.Context,
	userPublicID string,
	provider string,
	subject string,
	email string,
) error {
	result, err := r.upsertIdentityStmt.ExecContext(ctx, userPublicID, provider, subject, email)
	if err != nil {
		return fmt.Errorf("upsert identity: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("upsert identity rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateMagicLinkToken stores one single-use passwordless sign-in token.
func (r *SQLRepository) CreateMagicLinkToken(ctx context.Context, token MagicLinkToken) error {
	if _, err := r.createTokenStmt.ExecContext(
		ctx,
		token.UserPublicID,
		token.TokenHash,
		token.ExpiresAt,
	); err != nil {
		return fmt.Errorf("create magic link token: %w", err)
	}
	return nil
}

// ConsumeMagicLinkToken atomically marks a valid token as used and returns the
// account it belongs to. ErrNotFound is returned for unknown, used, or expired
// tokens.
func (r *SQLRepository) ConsumeMagicLinkToken(
	ctx context.Context,
	tokenHash string,
) (string, error) {
	var publicID string
	err := r.consumeTokenStmt.GetContext(ctx, &publicID, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("consume magic link token: %w", err)
	}
	return publicID, nil
}

// TouchLastLogin records a successful sign-in timestamp.
func (r *SQLRepository) TouchLastLogin(ctx context.Context, publicID string) error {
	if _, err := r.touchStmt.ExecContext(ctx, publicID); err != nil {
		return fmt.Errorf("touch last login: %w", err)
	}
	return nil
}

// CreatePasswordResetToken stores one single-use password reset token.
func (r *SQLRepository) CreatePasswordResetToken(
	ctx context.Context,
	token PasswordResetToken,
) error {
	if _, err := r.createResetStmt.ExecContext(
		ctx,
		token.UserPublicID,
		token.TokenHash,
		token.ExpiresAt,
	); err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	return nil
}

// ConsumePasswordResetToken atomically marks a valid reset token as used and
// returns the account it belongs to.
func (r *SQLRepository) ConsumePasswordResetToken(
	ctx context.Context,
	tokenHash string,
) (string, error) {
	var publicID string
	err := r.consumeResetStmt.GetContext(ctx, &publicID, tokenHash)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("consume password reset token: %w", err)
	}
	return publicID, nil
}

// UpdatePasswordHash stores a new password hash and clears the invited marker.
func (r *SQLRepository) UpdatePasswordHash(
	ctx context.Context,
	publicID string,
	hash string,
) error {
	result, err := r.updatePassStmt.ExecContext(ctx, publicID, hash)
	if err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update password hash rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
