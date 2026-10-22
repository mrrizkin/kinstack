package notifications

import (
	"context"
	"errors"
	"fmt"
	"testing"
)

func newTestService() (*Service, *MemoryRepository) {
	repository := NewMemoryRepository()
	return NewService(ServiceDependency{Repository: repository}), repository
}

func TestNotifyCreatesUnreadNotification(t *testing.T) {
	service, _ := newTestService()
	created, err := service.Notify(
		context.Background(),
		"user-1",
		"system",
		"Halo",
		"Pesan pertama",
	)
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

	props, err := service.List(context.Background(), "user-1", "", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(props.Notifications) != 1 || props.Unread != 1 {
		t.Fatalf("unexpected props: %#v", props)
	}
	if props.NextCursor != "" {
		t.Fatalf("expected last chunk, got cursor %q", props.NextCursor)
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
	props, err := service.List(context.Background(), "user-1", "", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if props.Unread != 0 {
		t.Fatalf("expected no unread, got %d", props.Unread)
	}
}

func TestListPagesByKeysetWithoutRepeats(t *testing.T) {
	service, repository := newTestService()
	want := make([]string, 0, 7)
	for index := 1; index <= 7; index++ {
		publicID := fmt.Sprintf("n-%d", index)
		want = append(want, publicID)
		SeedDefault(repository, "user-1", publicID, publicID)
	}
	SeedDefault(repository, "user-2", "n-other", "Other")

	seen := make([]string, 0, len(want))
	cursor := ""
	chunks := 0
	for {
		props, err := service.List(context.Background(), "user-1", cursor, 3)
		if err != nil {
			t.Fatalf("list chunk %d: %v", chunks, err)
		}
		if len(props.Notifications) > 3 {
			t.Fatalf("chunk exceeds the limit: %d", len(props.Notifications))
		}
		for _, item := range props.Notifications {
			seen = append(seen, item.PublicID)
		}
		chunks++
		if props.NextCursor == "" {
			break
		}
		cursor = props.NextCursor
		if chunks > len(want)+1 {
			t.Fatal("pagination did not terminate")
		}
	}

	if chunks != 3 {
		t.Fatalf("expected 3 chunks, got %d", chunks)
	}
	if len(seen) != len(want) {
		t.Fatalf("expected %d rows, got %d", len(want), len(seen))
	}
	unique := make(map[string]bool, len(seen))
	for index, publicID := range seen {
		if unique[publicID] {
			t.Fatalf("notification %q repeated across chunks", publicID)
		}
		unique[publicID] = true
		if expected := want[len(want)-1-index]; publicID != expected {
			t.Fatalf("row %d = %q, want %q", index, publicID, expected)
		}
	}
}

func TestListRejectsMalformedCursor(t *testing.T) {
	service, repository := newTestService()
	SeedDefault(repository, "user-1", "n-1", "Satu")

	for _, cursor := range []string{"nope", "abc:1", "1:abc"} {
		if _, err := service.List(
			context.Background(),
			"user-1",
			cursor,
			10,
		); !errors.Is(
			err,
			ErrInvalidInput,
		) {
			t.Fatalf("cursor %q: expected ErrInvalidInput, got %v", cursor, err)
		}
	}
}
