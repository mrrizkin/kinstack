package sessionregistry

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

var (
	// ErrNotFound identifies a session that is absent or already revoked.
	ErrNotFound = errors.New(i18n.T("session.not_found"))
	// ErrInvalidInput identifies malformed session registration input.
	ErrInvalidInput = errors.New(i18n.T("session.invalid_input"))
	// ErrRepositoryUnavailable identifies a session registry without a repository.
	ErrRepositoryUnavailable = errors.New("session registry repository is not configured")
)
