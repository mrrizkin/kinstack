package media

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

var (
	// ErrNotFound identifies a missing media asset.
	ErrNotFound = errors.New(i18n.T("media.not_found"))
	// ErrInvalidInput identifies an unsupported or malformed upload.
	ErrInvalidInput = errors.New(i18n.T("media.invalid_input"))
	// ErrStorage identifies an object storage failure.
	ErrStorage = errors.New(i18n.T("media.storage_unavailable"))
)
