package users

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

var (
	ErrNotFound      = errors.New(i18n.T("users.not_found"))
	ErrUsernameTaken = errors.New(i18n.T("users.username_taken"))
	ErrEmailTaken    = errors.New(i18n.T("users.email_taken"))
	ErrInvalidInput  = errors.New(i18n.T("users.invalid_input"))
	ErrInvalidRole   = errors.New(i18n.T("users.invalid_role"))
	// ErrLastAdministrator prevents disabling the last account that can manage access.
	ErrLastAdministrator        = errors.New(i18n.T("users.last_administrator"))
	ErrAuthorizationUnavailable = errors.New(i18n.T("users.authorization_unavail"))
	// ErrForbidden is a change that would exceed the actor's own authority.
	ErrForbidden  = errors.New(i18n.T("authz.reason.privilege_escalation"))
	ErrSelfChange = errors.New(i18n.T("authz.reason.self_change"))
	ErrSystemRole = errors.New(i18n.T("authz.reason.system_role"))
)
