package actor

import (
	"context"
	"errors"
	"testing"
)

func TestActorValidatesActiveActors(t *testing.T) {
	repository := NewMemoryRepository("actor-1")
	actor := New(Dependency{Repository: repository})

	if err := actor.Validate(context.Background(), " actor-1 "); err != nil {
		t.Fatalf("validate actor: %v", err)
	}

	repository.Remove("actor-1")
	if err := actor.Validate(context.Background(), "actor-1"); !errors.Is(err, ErrInvalidActor) {
		t.Fatalf("expected invalid actor error, got %v", err)
	}

	repository.Add("actor-1")
	if err := actor.Validate(context.Background(), "actor-1"); err != nil {
		t.Fatalf("validate restored actor: %v", err)
	}
}

func TestNewActorRequiresRepository(t *testing.T) {
	deferred := false
	defer func() {
		if !deferred {
			t.Fatal("expected actor constructor to panic")
		}
	}()
	defer func() {
		deferred = recover() != nil
	}()
	New(Dependency{})
}

func TestActorRejectsEmptyPublicID(t *testing.T) {
	actor := New(Dependency{Repository: NewMemoryRepository("actor-1")})
	if err := actor.Validate(context.Background(), " "); !errors.Is(err, ErrInvalidActor) {
		t.Fatalf("expected invalid actor error, got %v", err)
	}
}

func TestValidateSessionTrustsSuperuser(t *testing.T) {
	repository := NewMemoryRepository()
	actor := New(Dependency{Repository: repository})

	session := &SessionActor{PublicID: SuperuserPublicID, Superuser: true}
	if err := actor.ValidateSession(context.Background(), session); err != nil {
		t.Fatalf("superuser session rejected: %v", err)
	}
	if err := actor.ValidateSession(context.Background(), nil); !errors.Is(err, ErrInvalidActor) {
		t.Fatalf("expected invalid actor error, got %v", err)
	}
}
