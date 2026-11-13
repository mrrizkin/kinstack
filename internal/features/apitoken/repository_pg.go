package apitoken

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	apitokensqlc "kinstack/internal/features/apitoken/sqlc"
	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/pagination"

	"github.com/lib/pq"
)

type SQLRepository struct {
	queries *apitokensqlc.Queries
}

var _ Repository = (*SQLRepository)(nil)

type RepositoryDependency struct {
	DB *sql.DB
}

func NewSQLRepository(dependency RepositoryDependency) (*SQLRepository, error) {
	assert.Assert(dependency.DB != nil, "api token database is required")

	queries, err := apitokensqlc.Prepare(context.Background(), dependency.DB)
	if err != nil {
		return nil, fmt.Errorf("prepare api token queries: %w", err)
	}
	return &SQLRepository{queries: queries}, nil
}

func (r *SQLRepository) Close() error {
	if r == nil {
		return nil
	}
	return r.queries.Close()
}

func (r *SQLRepository) Create(
	ctx context.Context,
	input CreateTokenInput,
) (TokenResult, error) {
	row, err := r.queries.CreateToken(ctx, apitokensqlc.CreateTokenParams{
		PublicID:     input.PublicID,
		UserPublicID: input.UserPublicID,
		Name:         input.Name,
		TokenHash:    input.TokenHash,
		Prefix:       input.Prefix,
		Scopes:       []string(input.Scopes),
		ExpiresAt:    nullTime(input.ExpiresAt),
	})
	if err != nil {
		return TokenResult{}, fmt.Errorf("create api token: %w", err)
	}
	return tokenFromCreateRow(row), nil
}

func (r *SQLRepository) ListByUser(
	ctx context.Context,
	userPublicID string,
	query pagination.Query,
) ([]TokenResult, error) {
	rows, err := r.queries.ListTokensByUser(ctx, apitokensqlc.ListTokensByUserParams{
		UserPublicID: userPublicID,
		RowLimit:     int32(query.Limit()),
		RowOffset:    int32(query.Offset()),
	})
	if err != nil {
		return nil, fmt.Errorf("list api tokens: %w", err)
	}
	results := make([]TokenResult, 0, len(rows))
	for _, row := range rows {
		results = append(results, tokenFromListRow(row))
	}
	return results, nil
}

func (r *SQLRepository) CountByUser(ctx context.Context, userPublicID string) (int, error) {
	count, err := r.queries.CountTokensByUser(ctx, userPublicID)
	if err != nil {
		return 0, fmt.Errorf("count api tokens: %w", err)
	}
	return int(count), nil
}

func (r *SQLRepository) FindByPublicID(ctx context.Context, publicID string) (TokenResult, error) {
	row, err := r.queries.FindTokenByPublicID(ctx, publicID)
	if errors.Is(err, sql.ErrNoRows) {
		return TokenResult{}, ErrNotFound
	}
	if err != nil {
		return TokenResult{}, fmt.Errorf("find api token: %w", err)
	}
	return tokenFromFindRow(row), nil
}

func (r *SQLRepository) Revoke(ctx context.Context, userPublicID, publicID string) error {
	affected, err := r.queries.RevokeToken(ctx, apitokensqlc.RevokeTokenParams{
		PublicID:     publicID,
		UserPublicID: userPublicID,
	})
	if err != nil {
		return fmt.Errorf("revoke api token: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

// RevokeAll revokes every active token of one user after a credential change.
func (r *SQLRepository) RevokeAll(ctx context.Context, userPublicID string) (int, error) {
	affected, err := r.queries.RevokeTokensByUser(ctx, userPublicID)
	if err != nil {
		return 0, fmt.Errorf("revoke all api tokens: %w", err)
	}
	return int(affected), nil
}

func (r *SQLRepository) TouchLastUsed(ctx context.Context, publicID string) error {
	if err := r.queries.TouchTokenLastUsed(ctx, publicID); err != nil {
		return fmt.Errorf("touch api token: %w", err)
	}
	return nil
}

func tokenFromCreateRow(row apitokensqlc.CreateTokenRow) TokenResult {
	return TokenResult{
		PublicID:     row.PublicID,
		UserPublicID: row.UserPublicID,
		Name:         row.Name,
		TokenHash:    row.TokenHash,
		Prefix:       row.Prefix,
		Scopes:       pq.StringArray(row.Scopes),
		ExpiresAt:    nullableTime(row.ExpiresAt),
		LastUsedAt:   nullableTime(row.LastUsedAt),
		RevokedAt:    nullableTime(row.RevokedAt),
		CreatedAt:    row.CreatedAt,
	}
}

func tokenFromListRow(row apitokensqlc.ListTokensByUserRow) TokenResult {
	return TokenResult{
		PublicID:     row.PublicID,
		UserPublicID: row.UserPublicID,
		Name:         row.Name,
		TokenHash:    row.TokenHash,
		Prefix:       row.Prefix,
		Scopes:       pq.StringArray(row.Scopes),
		ExpiresAt:    nullableTime(row.ExpiresAt),
		LastUsedAt:   nullableTime(row.LastUsedAt),
		RevokedAt:    nullableTime(row.RevokedAt),
		CreatedAt:    row.CreatedAt,
	}
}

func tokenFromFindRow(row apitokensqlc.FindTokenByPublicIDRow) TokenResult {
	return TokenResult{
		PublicID:     row.PublicID,
		UserPublicID: row.UserPublicID,
		Name:         row.Name,
		TokenHash:    row.TokenHash,
		Prefix:       row.Prefix,
		Scopes:       pq.StringArray(row.Scopes),
		ExpiresAt:    nullableTime(row.ExpiresAt),
		LastUsedAt:   nullableTime(row.LastUsedAt),
		RevokedAt:    nullableTime(row.RevokedAt),
		CreatedAt:    row.CreatedAt,
	}
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
