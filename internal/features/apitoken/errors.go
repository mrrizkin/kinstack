package apitoken

import (
	"errors"

	"kinstack/internal/platform/i18n"
)

var (
	// ErrNotFound identifies an unknown API token.
	ErrNotFound = errors.New(i18n.T("apitoken.not_found"))
	// ErrInvalidToken identifies a malformed, revoked, or expired bearer token.
	ErrInvalidToken = errors.New(i18n.T("apitoken.invalid_token"))
	// ErrInvalidInput identifies a malformed token creation payload.
	ErrInvalidInput = errors.New(i18n.T("apitoken.invalid_input"))
)
