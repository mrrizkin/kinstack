package twofactor

import "context"

// StatusResult is the persistence projection of an account's two-factor state.
type StatusResult struct {
	Enabled bool   `db:"two_factor_enabled"`
	Secret  string `db:"two_factor_secret"`
}

// Repository is the persistence contract owned by the two-factor slice.
type Repository interface {
	FindStatus(context.Context, string) (StatusResult, error)
	SetSecret(context.Context, string, string) error
	Enable(context.Context, string) error
	Disable(context.Context, string) error
	ReplaceRecoveryCodes(context.Context, string, []string) error
	FindRecoveryCode(context.Context, string, string) (int64, error)
	MarkRecoveryCodeUsed(context.Context, int64) error
}
