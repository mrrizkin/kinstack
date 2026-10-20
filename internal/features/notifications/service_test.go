package notifications

import (
	"context"
	"errors"
	"testing"
)

func newTestService() (*Service, *MemoryRepository) {
	repository := NewMemoryRepository()
	return NewService(ServiceDependency{Repository: repository}), repository
}

func TestNotifyCreatesUnreadNotification(t *testing.T) {
	service, _ := newTestService()
	created, err := service.Notify(context.Background(), "user-1", "system", "Halo", "Pesan pertama")
	if err != nil {
		t.Fatalf("notify: %v", err)
	}
	if created.PublicID == "" || created.Read || created.Title != "Halo" {
		t.Fatalf("unexpected notification: %#v", created)
	}
}

func TestNotifyRejectsEmptyFields(t *testing.T) {
	service, _ := newTestService()
	if _, err := service.Notify(context.Background(), "user-1", "", "Halo", "Pesan"); !errors.Is(
		err,
		ErrInvalidInput,
	) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestListReturnsUnreadCountAndScopesToRecipient(t *testing.T) {
	service, repository := newTestService()
	SeedDefault(repository, "user-1", "n-1", "Satu")
	SeedDefault(repository, "user-2", "n-2", "Dua")

	props, err := service.List(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(props.Notifications) != 1 || props.Unread != 1 {
		t.Fatalf("unexpected props: %#v", props)
	}
}

func TestMarkReadClearsUnread(t *testing.T) {
	service, repository := newTestService()
	SeedDefault(repository, "user-1", "n-1", "Satu")

	if err := service.MarkRead(context.Background(), "user-1", "n-1"); err != nil {
		t.Fatalf("mark read: %v", err)
	}
	unread, err := repository.UnreadCount(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("unread count: %v", err)
	}
	if unread != 0 {
		t.Fatalf("expected no unread, got %d", unread)
	}
}

func TestMarkReadRejectsForeignRecipient(t *testing.T) {
	service, repository := newTestService()
	SeedDefault(repository, "user-1", "n-1", "Satu")

	if err := service.MarkRead(context.Background(), "user-2", "n-1"); !errors.Is(
		err,
		ErrNotFound,
	) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestMarkAllRead(t *testing.T) {
	service, repository := newTestService()
	SeedDefault(repository, "user-1", "n-1", "Satu")
	SeedDefault(repository, "user-1", "n-2", "Dua")

	if err := service.MarkAllRead(context.Background(), "user-1"); err != nil {
		t.Fatalf("mark all: %v", err)
	}
	props, err := service.List(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if props.Unread != 0 {
		t.Fatalf("expected no unread, got %d", props.Unread)
	}
}
