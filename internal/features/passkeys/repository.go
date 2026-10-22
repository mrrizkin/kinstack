package passkeys

import (
	"context"
	"time"
)

// FindPasskeyUserResult contains the account fields needed by a passkey flow.
type FindPasskeyUserResult struct {
	PublicID    string     `db:"public_id"`
	Email       string     `db:"email"`
	Username    string     `db:"username"`
	DisplayName string     `db:"display_name"`
	IsActive    bool       `db:"is_active"`
	DeletedAt   *time.Time `db:"deleted_at"`
}

// ListPasskeysResult contains only the columns needed to render or verify credentials.
type ListPasskeysResult struct {
	ID         []byte     `db:"credential_id"`
	Name       string     `db:"name"`
	Credential []byte     `db:"credential"`
	CreatedAt  time.Time  `db:"created_at"`
	LastUsedAt *time.Time `db:"last_used_at"`
}

type Repository interface {
	FindByPublicID(context.Context, string) (FindPasskeyUserResult, error)
	ListPasskeys(context.Context, string) ([]ListPasskeysResult, error)
	SavePasskey(context.Context, string, PasskeyRecord) error
	UpdatePasskey(context.Context, string, PasskeyRecord) error
	DeletePasskey(context.Context, string, []byte) error
	FindPasskeyUser(
		context.Context,
		[]byte,
		[]byte,
	) (FindPasskeyUserResult, []ListPasskeysResult, error)
}
