// Package sessionregistry tracks the signed-in browser sessions of each user so
// they can be listed and revoked from the profile security page.
package sessionregistry

import (
	"context"
	"strings"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/publicid"
)

type Dependency struct {
	Repository Repository
	Store      SessionStore
}

// SessionStore destroys a server-side session by its transport identifier.
type SessionStore interface {
	Delete(ctx context.Context, id string) error
}

type Service struct {
	repository Repository
	store      SessionStore
}

func New(d Dependency) *Service {
	assert.Assert(d.Repository != nil, "session registry repository is required")

	return &Service{repository: d.Repository, store: d.Store}
}

// RegisterInput describes a freshly authenticated browser session.
type RegisterInput struct {
	UserPublicID string
	SessionID    string
	IP           string
	UserAgent    string
}

// Register records a newly authenticated browser session.
func (s *Service) Register(ctx context.Context, input RegisterInput) (Result, error) {
	sessionID := strings.TrimSpace(input.SessionID)
	if sessionID == "" || strings.TrimSpace(input.UserPublicID) == "" {
		return Result{}, ErrInvalidInput
	}
	return s.repository.Register(ctx, publicid.MustNew(), RegisterInput{
		UserPublicID: input.UserPublicID,
		SessionID:    sessionID,
		IP:           strings.TrimSpace(input.IP),
		UserAgent:    strings.TrimSpace(input.UserAgent),
	})
}

// Touch refreshes the last-seen timestamp without writing on every request.
func (s *Service) Touch(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	return s.repository.Touch(ctx, sessionID)
}

// List returns the active sessions of one user, most recent first.
func (s *Service) List(ctx context.Context, userPublicID string) ([]Result, error) {
	return s.repository.ListActive(ctx, strings.TrimSpace(userPublicID))
}

// Revoke destroys one session owned by the user. The transport session is
// removed from the store so the device is signed out immediately.
func (s *Service) Revoke(ctx context.Context, userPublicID, sessionPublicID string) error {
	result, err := s.repository.FindByPublicID(ctx, strings.TrimSpace(sessionPublicID))
	if err != nil {
		return err
	}
	if result.UserPublicID != strings.TrimSpace(userPublicID) {
		return ErrNotFound
	}
	if err := s.repository.Revoke(ctx, result.PublicID); err != nil {
		return err
	}
	s.destroy(ctx, result.SessionID)
	return nil
}

// RevokeOthers signs out every session except the current one.
func (s *Service) RevokeOthers(ctx context.Context, userPublicID, keepSessionID string) error {
	sessionIDs, err := s.repository.RevokeAllExcept(
		ctx,
		strings.TrimSpace(userPublicID),
		strings.TrimSpace(keepSessionID),
	)
	if err != nil {
		return err
	}
	for _, sessionID := range sessionIDs {
		s.destroy(ctx, sessionID)
	}
	return nil
}

// RevokeAll signs out every session of one user. It is used after a credential
// change so a stolen session cannot outlive the password.
func (s *Service) RevokeAll(ctx context.Context, userPublicID string) error {
	userPublicID = strings.TrimSpace(userPublicID)
	if userPublicID == "" {
		return nil
	}
	sessionIDs, err := s.repository.RevokeAll(ctx, userPublicID)
	if err != nil {
		return err
	}
	for _, sessionID := range sessionIDs {
		s.destroy(ctx, sessionID)
	}
	return nil
}

// RevokeSession marks one transport session as revoked. Used on logout, where
// the session store is already destroyed by the actor boundary.
func (s *Service) RevokeSession(ctx context.Context, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil
	}
	return s.repository.RevokeBySessionID(ctx, sessionID)
}

// destroy removes the transport session, ignoring a missing store so that
// revoked bookkeeping never fails because the store entry already expired.
func (s *Service) destroy(ctx context.Context, sessionID string) {
	if s.store == nil || sessionID == "" {
		return
	}
	_ = s.store.Delete(ctx, sessionID)
}
