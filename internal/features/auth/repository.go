package auth

import (
	"context"
	"time"
)

// FindUserByUsernameResult is the credential projection for one account.
type FindUserByUsernameResult struct {
	PublicID     string `db:"public_id"`
	Username     string `db:"username"`
	Email        string `db:"email"`
	DisplayName  string `db:"display_name"`
	PasswordHash string `db:"password_hash"`
	Role         string `db:"role"`
	IsActive     bool   `db:"is_active"`
}

// MagicLinkToken is a single-use passwordless sign-in token pending delivery.
type MagicLinkToken struct {
	UserPublicID string
	TokenHash    string
	ExpiresAt    time.Time
}

// PasswordResetToken is a single-use password reset token pending delivery.
type PasswordResetToken struct {
	UserPublicID string
	TokenHash    string
	ExpiresAt    time.Time
}

// Repository is the persistence contract owned by the auth slice.
type Repository interface {
	FindUserByUsername(context.Context, string) (FindUserByUsernameResult, error)
	FindUserByPublicID(context.Context, string) (FindUserByUsernameResult, error)
	FindUserByEmail(context.Context, string) (FindUserByUsernameResult, error)
	TouchLastLogin(context.Context, string) error
	CreateMagicLinkToken(context.Context, MagicLinkToken) error
	ConsumeMagicLinkToken(context.Context, string) (string, error)
	CreatePasswordResetToken(context.Context, PasswordResetToken) error
	ConsumePasswordResetToken(context.Context, string) (string, error)
	UpdatePasswordHash(context.Context, string, string) error
}
