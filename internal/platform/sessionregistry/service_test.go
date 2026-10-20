package sessionregistry

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type fakeStore struct {
	mu      sync.Mutex
	deleted []string
}

func (s *fakeStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleted = append(s.deleted, id)
	return nil
}

func newService(t *testing.T) (*Service, *MemoryRepository, *fakeStore) {
	t.Helper()
	repository := NewMemoryRepository("user-1")
	store := &fakeStore{}
	return New(Dependency{Repository: repository, Store: store}), repository, store
}

func TestRegisterListAndTouch(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	if _, err := service.Register(ctx, RegisterInput{
		UserPublicID: "user-1",
		SessionID:    "session-a",
		IP:           "10.0.0.1",
		UserAgent:    "Firefox",
	}); err != nil {
		t.Fatalf("register first session: %v", err)
	}
	if _, err := service.Register(ctx, RegisterInput{
		UserPublicID: "user-1",
		SessionID:    "session-b",
		IP:           "10.0.0.2",
		UserAgent:    "Chrome",
	}); err != nil {
		t.Fatalf("register second session: %v", err)
	}

	sessions, err := service.List(ctx, "user-1")
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	if err := service.Touch(ctx, "session-a"); err != nil {
		t.Fatalf("touch session: %v", err)
	}

	if _, err := service.Register(ctx, RegisterInput{
		UserPublicID: "missing",
		SessionID:    "session-x",
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for unknown user, got %v", err)
	}
}

func TestRevokeDestroysTransportSession(t *testing.T) {
	service, _, store := newService(t)
	ctx := context.Background()

	first, err := service.Register(ctx, RegisterInput{UserPublicID: "user-1", SessionID: "session-a"})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if _, err := service.Register(ctx, RegisterInput{UserPublicID: "user-1", SessionID: "session-b"}); err != nil {
		t.Fatalf("register: %v", err)
	}

	if err := service.Revoke(ctx, "user-1", first.PublicID); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if len(store.deleted) != 1 || store.deleted[0] != "session-a" {
		t.Fatalf("expected transport session to be destroyed, got %#v", store.deleted)
	}

	sessions, err := service.List(ctx, "user-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 1 || sessions[0].SessionID != "session-b" {
		t.Fatalf("unexpected remaining sessions: %#v", sessions)
	}
}

func TestRevokeRejectsForeignSession(t *testing.T) {
	service, repository, _ := newService(t)
	ctx := context.Background()

	if _, err := repository.Register(ctx, "session-public", RegisterInput{
		UserPublicID: "user-1",
		SessionID:    "session-a",
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := service.Revoke(ctx, "user-2", "session-public"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound for foreign session, got %v", err)
	}
}

func TestRevokeOthersKeepsCurrent(t *testing.T) {
	service, _, store := newService(t)
	ctx := context.Background()

	for _, sessionID := range []string{"session-a", "session-b", "session-c"} {
		if _, err := service.Register(ctx, RegisterInput{UserPublicID: "user-1", SessionID: sessionID}); err != nil {
			t.Fatalf("register %s: %v", sessionID, err)
		}
	}

	if err := service.RevokeOthers(ctx, "user-1", "session-b"); err != nil {
		t.Fatalf("revoke others: %v", err)
	}

	sessions, err := service.List(ctx, "user-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 1 || sessions[0].SessionID != "session-b" {
		t.Fatalf("expected only the current session, got %#v", sessions)
	}
	if len(store.deleted) != 2 {
		t.Fatalf("expected two transport sessions destroyed, got %#v", store.deleted)
	}
}

func TestRevokeSessionMarksLogout(t *testing.T) {
	service, _, _ := newService(t)
	ctx := context.Background()

	if _, err := service.Register(ctx, RegisterInput{UserPublicID: "user-1", SessionID: "session-a"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := service.RevokeSession(ctx, "session-a"); err != nil {
		t.Fatalf("revoke session: %v", err)
	}

	sessions, err := service.List(ctx, "user-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("expected no active sessions, got %#v", sessions)
	}
}
