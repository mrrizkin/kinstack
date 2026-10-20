package twofactor

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

var (
	// ErrNotFound identifies a missing user account.
	ErrNotFound = errors.New(i18n.T("twofactor.not_found"))
	// ErrInvalidInput identifies malformed two-factor input.
	ErrInvalidInput = errors.New(i18n.T("twofactor.invalid_input"))
	// ErrInvalidCode identifies a wrong TOTP or recovery code.
	ErrInvalidCode = errors.New(i18n.T("twofactor.invalid_code"))
	// ErrAlreadyEnabled identifies an account that already has two-factor auth.
	ErrAlreadyEnabled = errors.New(i18n.T("twofactor.already_enabled"))
	// ErrNotEnabled identifies an account without two-factor auth.
	ErrNotEnabled = errors.New(i18n.T("twofactor.not_enabled"))
	// ErrNotConfigured identifies a confirmation before enrollment started.
	ErrNotConfigured = errors.New(i18n.T("twofactor.not_configured"))
)
