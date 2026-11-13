package audit

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

var ErrInvalidInput = errors.New(i18n.T("audit.invalid_filter"))
