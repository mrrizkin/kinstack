package auth

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

var (
	// ErrInvalidCredentials identifies a failed username/password combination.
	ErrInvalidCredentials = errors.New(i18n.T("auth.invalid_credentials"))
	// ErrInactiveUser identifies a disabled account.
	ErrInactiveUser = errors.New(i18n.T("auth.inactive_user"))
	// ErrNotFound identifies a missing user record.
	ErrNotFound = errors.New(i18n.T("auth.user_not_found"))
	// ErrInvalidInput identifies malformed login input.
	ErrInvalidInput = errors.New(i18n.T("auth.invalid_input"))
	// ErrAccountLocked identifies a temporarily locked account.
	ErrAccountLocked = errors.New(i18n.T("auth.account_locked"))
	// ErrInvalidToken identifies an unknown, used, or expired sign-in token.
	ErrInvalidToken = errors.New(i18n.T("auth.invalid_token"))
	// ErrIdentityConflict identifies a provider subject already bound to another
	// account. It is reported as unregistered so the response cannot be used to
	// enumerate bindings.
	ErrIdentityConflict = errors.New("identity is already bound to another account")
	// errSSOStorageUnavailable identifies an SSO transaction without the
	// shared storage required to complete a cross-site callback.
	errSSOStorageUnavailable = errors.New("sso state storage is unavailable")
)
