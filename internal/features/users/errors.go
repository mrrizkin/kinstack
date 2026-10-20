package users

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

var (
	// ErrNotFound identifies a missing user account.
	ErrNotFound = errors.New(i18n.T("users.not_found"))
	// ErrUsernameTaken identifies a duplicate username.
	ErrUsernameTaken = errors.New(i18n.T("users.username_taken"))
	// ErrEmailTaken identifies a duplicate email address.
	ErrEmailTaken = errors.New(i18n.T("users.email_taken"))
	// ErrInvalidInput identifies malformed user input.
	ErrInvalidInput = errors.New(i18n.T("users.invalid_input"))
	// ErrInvalidRole identifies an unknown role.
	ErrInvalidRole = errors.New(i18n.T("users.invalid_role"))
	// ErrLastAdministrator prevents disabling the last account that can manage access.
	ErrLastAdministrator = errors.New(i18n.T("users.last_administrator"))
	// ErrAuthorizationUnavailable identifies missing authorization wiring.
	ErrAuthorizationUnavailable = errors.New(i18n.T("users.authorization_unavail"))
)
