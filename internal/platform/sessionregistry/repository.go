package sessionregistry

import (
	"context"
	"time"
)

type Result struct {
	PublicID     string     `db:"public_id"`
	UserPublicID string     `db:"user_public_id"`
	SessionID    string     `db:"session_id"`
	IP           string     `db:"ip"`
	UserAgent    string     `db:"user_agent"`
	CreatedAt    time.Time  `db:"created_at"`
	LastSeenAt   time.Time  `db:"last_seen_at"`
	RevokedAt    *time.Time `db:"revoked_at"`
}

type Repository interface {
	Register(context.Context, string, RegisterInput) (Result, error)
	Touch(context.Context, string) error
	ListActive(context.Context, string) ([]Result, error)
	FindByPublicID(context.Context, string) (Result, error)
	Revoke(context.Context, string) error
	RevokeBySessionID(context.Context, string) error
	RevokeAllExcept(context.Context, string, string) ([]string, error)
	RevokeAll(context.Context, string) ([]string, error)
}
