package twofactor

import "context"

// Secret is one account's TOTP secret. Exactly one of Plain and Sealed is set
// when writing: Sealed carries an envelope-encrypted secret, Plain exists only
// for legacy rows and local wiring without an encryption key.
type Secret struct {
	Plain      string
	Sealed     []byte
	KeyVersion int
}

// StatusResult is the persistence projection of an account's two-factor state.
type StatusResult struct {
	Enabled    bool   `db:"two_factor_enabled"`
	Secret     string `db:"two_factor_secret"`
	Sealed     []byte `db:"two_factor_secret_enc"`
	KeyVersion int    `db:"two_factor_key_version"`
	LastStep   int64  `db:"two_factor_last_step"`
}

// RecoveryCode is one stored one-time code hash.
type RecoveryCode struct {
	ID   int64  `db:"id"`
	Hash string `db:"code_hash"`
}

// LegacySecret is a plaintext TOTP secret that still needs sealing.
type LegacySecret struct {
	PublicID string `db:"public_id"`
	Secret   string `db:"two_factor_secret"`
}

// Repository is the persistence contract owned by the two-factor slice.
type Repository interface {
	FindStatus(context.Context, string) (StatusResult, error)
	SetSecret(context.Context, string, Secret) error
	SealSecret(context.Context, string, []byte, int) error
	Enable(context.Context, string, int64) error
	Disable(context.Context, string) error
	ConsumeStep(context.Context, string, int64) error
	ReplaceRecoveryCodes(context.Context, string, []string) error
	UnusedRecoveryCodes(context.Context, string) ([]RecoveryCode, error)
	MarkRecoveryCodeUsed(context.Context, int64) error
	ReplaceRecoveryCodeHash(context.Context, int64, string) error
	LegacySecrets(context.Context, int) ([]LegacySecret, error)
}
