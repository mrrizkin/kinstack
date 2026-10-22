// Package actor owns the shared authenticated-actor boundary.
package actor

import (
	"context"
	"errors"
	"strings"

	"kinstack/internal/pkg/assert"
)

// Actor validates authenticated actor identifiers through its own repository.
type Actor struct {
	repository Repository
}

type Dependency struct {
	Repository Repository
}

func New(d Dependency) *Actor {
	assert.Assert(d.Repository != nil, "actor repository is required")

	return &Actor{repository: d.Repository}
}

// Validate validates that an active account exists for publicID.
func (a *Actor) Validate(ctx context.Context, publicID string) error {
	publicID = strings.TrimSpace(publicID)
	if publicID == "" {
		return ErrInvalidActor
	}
	if err := a.repository.FindByPublicID(ctx, publicID); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidActor
		}
		return err
	}
	return nil
}
