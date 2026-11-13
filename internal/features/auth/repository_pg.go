package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	authsqlc "kinstack/internal/features/auth/sqlc"
	"kinstack/internal/pkg/assert"
)

type SQLRepository struct {
	queries *authsqlc.Queries
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "auth database is required")

	queries, err := authsqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare auth queries: %w", err)
	}
	return &SQLRepository{queries: queries}, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

func (r *SQLRepository) FindUserByUsername(
	ctx context.Context,
	username string,
) (FindUserByUsernameResult, error) {
	row, err := r.queries.FindUserByUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return FindUserByUsernameResult{}, ErrNotFound
	}
	if err != nil {
		return FindUserByUsernameResult{}, fmt.Errorf("find user by username: %w", err)
	}
	return resultFromUsernameRow(row), nil
}

func (r *SQLRepository) FindUserByPublicID(
	ctx context.Context,
	publicID string,
) (FindUserByUsernameResult, error) {
	row, err := r.queries.FindAuthUserByPublicID(ctx, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return FindUserByUsernameResult{}, ErrNotFound
	}
	if err != nil {
		return FindUserByUsernameResult{}, fmt.Errorf("find user by public ID: %w", err)
	}
	return resultFromPublicIDRow(row), nil
}

func (r *SQLRepository) FindUserByEmail(
	ctx context.Context,
	email string,
) (FindUserByUsernameResult, error) {
	row, err := r.queries.FindUserByEmail(ctx, email)
	if errors.Is(err, sql.ErrNoRows) {
		return FindUserByUsernameResult{}, ErrNotFound
	}
	if err != nil {
		return FindUserByUsernameResult{}, fmt.Errorf("find user by email: %w", err)
	}
	return resultFromEmailRow(row), nil
}

func (r *SQLRepository) FindIdentityUser(
	ctx context.Context,
	provider string,
	subject string,
) (FindUserByUsernameResult, error) {
	row, err := r.queries.FindIdentityUser(ctx, authsqlc.FindIdentityUserParams{
		Provider: provider,
		Subject:  subject,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return FindUserByUsernameResult{}, ErrNotFound
	}
	if err != nil {
		return FindUserByUsernameResult{}, fmt.Errorf("find identity user: %w", err)
	}
	return resultFromIdentityRow(row), nil
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
	affected, err := r.queries.UpsertIdentity(ctx, authsqlc.UpsertIdentityParams{
		UserPublicID: userPublicID,
		Provider:     provider,
		Subject:      subject,
		Email:        email,
	})
	if err != nil {
		return fmt.Errorf("upsert identity: %w", err)
	}
	if affected == 0 {
		// Either the account disappeared or the subject already belongs to a
		// different account; both are refusals, not internal failures.
		return ErrIdentityConflict
	}
	return nil
}

func (r *SQLRepository) TouchLastLogin(ctx context.Context, publicID string) error {
	if err := r.queries.TouchLastLogin(ctx, publicID); err != nil {
		return fmt.Errorf("touch last login: %w", err)
	}
	return nil
}

func (r *SQLRepository) CreatePasswordResetToken(
	ctx context.Context,
	token PasswordResetToken,
) error {
	if err := r.queries.CreatePasswordResetToken(ctx, authsqlc.CreatePasswordResetTokenParams{
		UserPublicID: token.UserPublicID,
		TokenHash:    token.TokenHash,
		ExpiresAt:    token.ExpiresAt,
	}); err != nil {
		return fmt.Errorf("create password reset token: %w", err)
	}
	return nil
}

func (r *SQLRepository) ConsumePasswordResetToken(
	ctx context.Context,
	tokenHash string,
) (string, error) {
	publicID, err := r.queries.ConsumePasswordResetToken(ctx, tokenHash)
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
	affected, err := r.queries.UpdatePasswordHash(ctx, authsqlc.UpdatePasswordHashParams{
		PublicID:     publicID,
		PasswordHash: hash,
	})
	if err != nil {
		return fmt.Errorf("update password hash: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func resultFromUsernameRow(row authsqlc.FindUserByUsernameRow) FindUserByUsernameResult {
	return FindUserByUsernameResult{
		PublicID:     row.PublicID,
		Username:     row.Username,
		Email:        row.Email,
		DisplayName:  row.DisplayName,
		PasswordHash: row.PasswordHash,
		Role:         row.Role,
		IsActive:     row.IsActive,
	}
}

func resultFromPublicIDRow(row authsqlc.FindAuthUserByPublicIDRow) FindUserByUsernameResult {
	return FindUserByUsernameResult{
		PublicID:     row.PublicID,
		Username:     row.Username,
		Email:        row.Email,
		DisplayName:  row.DisplayName,
		PasswordHash: row.PasswordHash,
		Role:         row.Role,
		IsActive:     row.IsActive,
	}
}

func resultFromEmailRow(row authsqlc.FindUserByEmailRow) FindUserByUsernameResult {
	return FindUserByUsernameResult{
		PublicID:     row.PublicID,
		Username:     row.Username,
		Email:        row.Email,
		DisplayName:  row.DisplayName,
		PasswordHash: row.PasswordHash,
		Role:         row.Role,
		IsActive:     row.IsActive,
	}
}

func resultFromIdentityRow(row authsqlc.FindIdentityUserRow) FindUserByUsernameResult {
	return FindUserByUsernameResult{
		PublicID:     row.PublicID,
		Username:     row.Username,
		Email:        row.Email,
		DisplayName:  row.DisplayName,
		PasswordHash: row.PasswordHash,
		Role:         row.Role,
		IsActive:     row.IsActive,
	}
}
