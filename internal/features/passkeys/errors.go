package passkeys

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

var (
	// ErrUserNotFound identifies an unknown or inactive account.
	ErrUserNotFound = errors.New(i18n.T("passkeys.user_not_found"))
	// ErrPasskeyUnavailable identifies a missing session or configuration.
	ErrPasskeyUnavailable = errors.New(i18n.T("passkeys.unavailable"))
	// ErrPasskeyNotFound identifies an unknown credential.
	ErrPasskeyNotFound = errors.New(i18n.T("passkeys.not_found"))
	// ErrPasskeyConflict identifies an already-registered credential.
	ErrPasskeyConflict = errors.New(i18n.T("passkeys.conflict"))
	// ErrPasskeyInvalid identifies a malformed credential or request.
	ErrPasskeyInvalid = errors.New(i18n.T("passkeys.invalid"))
)
