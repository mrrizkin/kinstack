package apitoken

import (
	"context"
	"time"

	"github.com/lib/pq"
)

// TokenResult is the persistence projection of one API token.
type TokenResult struct {
	PublicID     string         `db:"public_id"`
	UserPublicID string         `db:"user_public_id"`
	Name         string         `db:"name"`
	TokenHash    string         `db:"token_hash"`
	Prefix       string         `db:"prefix"`
	Scopes       pq.StringArray `db:"scopes"`
	ExpiresAt    *time.Time     `db:"expires_at"`
	LastUsedAt   *time.Time     `db:"last_used_at"`
	RevokedAt    *time.Time     `db:"revoked_at"`
	CreatedAt    time.Time      `db:"created_at"`
}

// CreateTokenInput is the persistence input for a new API token.
type CreateTokenInput struct {
	PublicID     string
	UserPublicID string
	Name         string
	TokenHash    string
	Prefix       string
	Scopes       pq.StringArray
	ExpiresAt    *time.Time
}

// Repository is the persistence contract owned by the apitoken feature.
type Repository interface {
	Create(context.Context, CreateTokenInput) (TokenResult, error)
	ListByUser(context.Context, string) ([]TokenResult, error)
	FindByPublicID(context.Context, string) (TokenResult, error)
	Revoke(context.Context, string, string) error
	RevokeAll(context.Context, string) (int, error)
	TouchLastUsed(context.Context, string) error
}
