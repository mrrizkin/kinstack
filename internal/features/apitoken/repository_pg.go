package apitoken

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"kinstack/internal/pkg/assert"
	platformdatabase "kinstack/internal/platform/database"

	"github.com/jmoiron/sqlx"
)

const tokenColumns = `
public_id,
user_public_id,
name,
token_hash,
prefix,
scopes,
expires_at,
last_used_at,
revoked_at,
created_at`

const (
	createTokenQuery = `
INSERT INTO api_tokens (public_id, user_public_id, name, token_hash, prefix, scopes, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING ` + tokenColumns

	listTokensQuery = `
SELECT ` + tokenColumns + `
FROM api_tokens
WHERE user_public_id = $1
  AND revoked_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT 100`

	findTokenQuery = `
SELECT ` + tokenColumns + `
FROM api_tokens
WHERE public_id = $1
LIMIT 1`

	revokeTokenQuery = `
UPDATE api_tokens
SET revoked_at = now()
WHERE public_id = $1
  AND user_public_id = $2
  AND revoked_at IS NULL`

	revokeAllTokensQuery = `
UPDATE api_tokens
SET revoked_at = now()
WHERE user_public_id = $1
  AND revoked_at IS NULL`

	touchTokenQuery = `
UPDATE api_tokens
SET last_used_at = now()
WHERE public_id = $1
  AND revoked_at IS NULL`
)

// SQLRepository is the PostgreSQL API token repository.
type SQLRepository struct {
	owner     *platformdatabase.StatementOwner
	create    *sqlx.Stmt
	list      *sqlx.Stmt
	find      *sqlx.Stmt
	revoke    *sqlx.Stmt
	revokeAll *sqlx.Stmt
	touch     *sqlx.Stmt
}

var _ Repository = (*SQLRepository)(nil)

// RepositoryDependency contains the dependencies required to build the repository.
type RepositoryDependency struct {
	DB *sqlx.DB
}

// NewSQLRepository constructs the PostgreSQL API token repository.
func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "api token database is required")

	owner := platformdatabase.NewStatementOwner(dependency.DB)
	repository := &SQLRepository{owner: owner}
	statements := []struct {
		target **sqlx.Stmt
		name   string
		query  string
	}{
		{&repository.create, "api_token_create", createTokenQuery},
		{&repository.list, "api_token_list", listTokensQuery},
		{&repository.find, "api_token_find", findTokenQuery},
		{&repository.revoke, "api_token_revoke", revokeTokenQuery},
		{&repository.revokeAll, "api_token_revoke_all", revokeAllTokensQuery},
		{&repository.touch, "api_token_touch", touchTokenQuery},
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

// Create stores a new API token.
func (r *SQLRepository) Create(
	ctx context.Context,
	input CreateTokenInput,
) (TokenResult, error) {
	var result TokenResult
	if err := r.create.GetContext(
		ctx,
		&result,
		input.PublicID,
		input.UserPublicID,
		input.Name,
		input.TokenHash,
		input.Prefix,
		input.Scopes,
		input.ExpiresAt,
	); err != nil {
		return TokenResult{}, fmt.Errorf("create api token: %w", err)
	}
	return result, nil
}

// ListByUser returns the active tokens of one user.
func (r *SQLRepository) ListByUser(
	ctx context.Context,
	userPublicID string,
) ([]TokenResult, error) {
	results := make([]TokenResult, 0)
	if err := r.list.SelectContext(ctx, &results, userPublicID); err != nil {
		return nil, fmt.Errorf("list api tokens: %w", err)
	}
	return results, nil
}

// FindByPublicID resolves one token by its public identifier.
func (r *SQLRepository) FindByPublicID(ctx context.Context, publicID string) (TokenResult, error) {
	var result TokenResult
	if err := r.find.GetContext(ctx, &result, publicID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TokenResult{}, ErrNotFound
		}
		return TokenResult{}, fmt.Errorf("find api token: %w", err)
	}
	return result, nil
}

// Revoke marks one token owned by the user as revoked.
func (r *SQLRepository) Revoke(ctx context.Context, userPublicID, publicID string) error {
	result, err := r.revoke.ExecContext(ctx, publicID, userPublicID)
	if err != nil {
		return fmt.Errorf("revoke api token: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke api token rows: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// RevokeAll revokes every active token of one user after a credential change.
func (r *SQLRepository) RevokeAll(ctx context.Context, userPublicID string) (int, error) {
	result, err := r.revokeAll.ExecContext(ctx, userPublicID)
	if err != nil {
		return 0, fmt.Errorf("revoke all api tokens: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("revoke all api token rows: %w", err)
	}
	return int(affected), nil
}

// TouchLastUsed records the last successful use of a token.
func (r *SQLRepository) TouchLastUsed(ctx context.Context, publicID string) error {
	if _, err := r.touch.ExecContext(ctx, publicID); err != nil {
		return fmt.Errorf("touch api token: %w", err)
	}
	return nil
}
