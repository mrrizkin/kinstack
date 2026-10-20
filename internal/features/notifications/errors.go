package notifications

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

var (
	// ErrNotFound identifies a missing notification.
	ErrNotFound = errors.New(i18n.T("notifications.not_found"))
	// ErrInvalidInput identifies a malformed notification.
	ErrInvalidInput = errors.New(i18n.T("notifications.invalid_input"))
)
