package actor

import "errors"

var (
	// ErrInvalidActor identifies an actor that does not resolve to an active account.
	ErrInvalidActor = errors.New("actor is invalid")
	// ErrNotFound identifies an actor that is absent from the repository.
	ErrNotFound = errors.New("actor not found")
	// ErrRepositoryUnavailable identifies an actor with no configured repository.
	ErrRepositoryUnavailable = errors.New("actor repository is not configured")
)
