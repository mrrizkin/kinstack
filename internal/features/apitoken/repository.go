package apitoken

import (
	"context"
	"time"

	"kinstack/internal/pkg/pagination"

	"github.com/lib/pq"
)

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

type CreateTokenInput struct {
	PublicID     string
	UserPublicID string
	Name         string
	TokenHash    string
	Prefix       string
	Scopes       pq.StringArray
	ExpiresAt    *time.Time
}

type Repository interface {
	Create(context.Context, CreateTokenInput) (TokenResult, error)
	ListByUser(context.Context, string, pagination.Query) ([]TokenResult, error)
	CountByUser(context.Context, string) (int, error)
	FindByPublicID(context.Context, string) (TokenResult, error)
	Revoke(context.Context, string, string) error
	RevokeAll(context.Context, string) (int, error)
	TouchLastUsed(context.Context, string) error
}
