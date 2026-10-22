package profile

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

var (
	// ErrNotFound identifies a missing user profile.
	ErrNotFound = errors.New(i18n.T("profile.not_found"))
	// ErrInvalidInput identifies malformed profile input.
	ErrInvalidInput = errors.New(i18n.T("profile.invalid_input"))
	// ErrWrongPassword identifies a wrong current password.
	ErrWrongPassword = errors.New(i18n.T("profile.wrong_password"))
	// ErrPasswordMismatch identifies a wrong password confirmation.
	ErrPasswordMismatch = errors.New(i18n.T("profile.password_mismatch"))
	// ErrInvalidTheme identifies an unsupported display theme.
	ErrInvalidTheme = errors.New(i18n.T("profile.theme_invalid"))
	// ErrInvalidPageSize identifies a page size the list views cannot render.
	ErrInvalidPageSize = errors.New(i18n.T("profile.page_size_invalid"))
)
