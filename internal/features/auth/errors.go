package auth

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

var (
	ErrInvalidCredentials = errors.New(i18n.T("auth.invalid_credentials"))
	ErrInactiveUser       = errors.New(i18n.T("auth.inactive_user"))
	ErrNotFound           = errors.New(i18n.T("auth.user_not_found"))
	ErrInvalidInput       = errors.New(i18n.T("auth.invalid_input"))
	ErrAccountLocked      = errors.New(i18n.T("auth.account_locked"))
	ErrInvalidToken       = errors.New(i18n.T("auth.invalid_token"))
	// ErrIdentityConflict is reported as unregistered, so the response cannot enumerate
	// bindings.
	ErrIdentityConflict      = errors.New("identity is already bound to another account")
	errSSOStorageUnavailable = errors.New("sso state storage is unavailable")
)
