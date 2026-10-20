package audit

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

// ErrInvalidInput identifies a malformed audit filter value.
var ErrInvalidInput = errors.New(i18n.T("audit.invalid_filter"))
