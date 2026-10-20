package passkeys

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

func TestServiceListsAndRevokesActorPasskeys(t *testing.T) {
	repository := NewMemoryRepository()
	service := NewService(ServiceDependency{Repository: repository})
	record := PasskeyRecord{
		ID: []byte("credential-service"), Name: "  Laptop  ", CreatedAt: time.Now().UTC(),
		Credential: webauthn.Credential{
			ID:        []byte("credential-service"),
			PublicKey: []byte("public-key"),
		},
	}
	if err := service.SavePasskey(
		context.Background(),
		"A1b2C3d4E5f6G7h8I9j0K",
		record,
	); err != nil {
		t.Fatalf("save passkey: %v", err)
	}

	items, err := service.ListPasskeys(context.Background(), "A1b2C3d4E5f6G7h8I9j0K")
	if err != nil {
		t.Fatalf("list passkeys: %v", err)
	}
	if len(items) != 1 || items[0].Name != "Laptop" || items[0].ID == "" ||
		items[0].CreatedAt == "" {
		t.Fatalf("unexpected passkey summary: %#v", items)
	}

	if err := service.DeletePasskey(
		context.Background(),
		"C1d2E3f4G5h6I7j8K9l0M",
		record.ID,
	); !errors.Is(err, ErrPasskeyNotFound) {
		t.Fatalf("cross-actor delete error=%v, want %v", err, ErrPasskeyNotFound)
	}
	if err := service.DeletePasskey(
		context.Background(),
		"A1b2C3d4E5f6G7h8I9j0K",
		record.ID,
	); err != nil {
		t.Fatalf("delete passkey: %v", err)
	}
	items, err = service.ListPasskeys(context.Background(), "A1b2C3d4E5f6G7h8I9j0K")
	if err != nil || len(items) != 0 {
		t.Fatalf("expected empty passkey list, items=%#v err=%v", items, err)
	}
}

func TestServiceRejectsInvalidPasskeyAndUnknownActor(t *testing.T) {
	service := NewService(ServiceDependency{Repository: NewMemoryRepository()})
	invalid := PasskeyRecord{
		ID:   []byte("credential"),
		Name: "",
		Credential: webauthn.Credential{
			ID: []byte("credential"),
		},
	}
	if err := service.SavePasskey(
		context.Background(),
		"A1b2C3d4E5f6G7h8I9j0K",
		invalid,
	); !errors.Is(err, ErrPasskeyInvalid) {
		t.Fatalf("invalid passkey error=%v, want %v", err, ErrPasskeyInvalid)
	}
	if _, err := service.ListPasskeys(context.Background(), "unknown"); !errors.Is(
		err,
		ErrUserNotFound,
	) {
		t.Fatalf("unknown actor error=%v, want %v", err, ErrUserNotFound)
	}
}
