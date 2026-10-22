package audit

import (
	"context"
	"errors"
	"testing"
	"time"

	"kinstack/internal/pkg/pagination"
)

func newSeededRepository() *MemoryRepository {
	repository := NewMemoryRepository()
	base := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	repository.Seed(Entry{
		OccurredAt: base,
		RequestID:  "req-read",
		ActorKey:   "actor-1",
		Action:     "http.read",
		Resource:   "/users",
		Method:     "GET",
		Status:     200,
		Outcome:    "success",
		Metadata:   map[string]string{"source": "http"},
	})
	repository.Seed(Entry{
		OccurredAt: base.Add(time.Minute),
		RequestID:  "req-mutate",
		ActorKey:   "actor-1",
		Action:     "http.mutate",
		Resource:   "/users/:publicID/active",
		Method:     "POST",
		Status:     409,
		Outcome:    "failure",
	})
	repository.Seed(Entry{
		OccurredAt: base.Add(2 * time.Minute),
		RequestID:  "req-other",
		ActorKey:   "actor-2",
		Action:     "http.read",
		Resource:   "/access",
		Method:     "GET",
		Status:     200,
		Outcome:    "success",
	})
	return repository
}

func newTestService() *Service {
	return NewService(ServiceDependency{Repository: newSeededRepository()})
}

func TestListReturnsNewestFirst(t *testing.T) {
	service := newTestService()
	props, err := service.List(context.Background(), FilterRequest{})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if props.Page.Total != 3 || len(props.Entries) != 3 {
		t.Fatalf("unexpected page: %#v", props)
	}
	if props.Entries[0].RequestID != "req-other" || props.Entries[2].RequestID != "req-read" {
		t.Fatalf("expected newest-first ordering: %#v", props.Entries)
	}
	if props.Page.PerPage != pagination.DefaultPerPage || props.Page.Page != 1 ||
		props.Page.TotalPages != 1 {
		t.Fatalf("unexpected pagination: %#v", props)
	}
}

func TestListFiltersByActorAndOutcome(t *testing.T) {
	service := newTestService()
	props, err := service.List(context.Background(), FilterRequest{
		ActorKey: "actor-1",
		Outcome:  "failure",
	})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if props.Page.Total != 1 || len(props.Entries) != 1 ||
		props.Entries[0].RequestID != "req-mutate" {
		t.Fatalf("unexpected filtered page: %#v", props)
	}
}

func TestListSearchMatchesResource(t *testing.T) {
	service := newTestService()
	props, err := service.List(context.Background(), FilterRequest{Search: "access"})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if props.Page.Total != 1 || props.Entries[0].ActorKey != "actor-2" {
		t.Fatalf("unexpected search page: %#v", props)
	}
}

func TestListPaginates(t *testing.T) {
	service := newTestService()
	props, err := service.List(context.Background(), FilterRequest{Page: 2, PerPage: 2})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if props.Page.Total != 3 || props.Page.TotalPages != 2 || props.Page.Page != 2 ||
		len(props.Entries) != 1 {
		t.Fatalf("unexpected pagination: %#v", props)
	}
}

func TestListRejectsUnknownFilters(t *testing.T) {
	service := newTestService()
	for name, request := range map[string]FilterRequest{
		"action":  {Action: "http.delete"},
		"outcome": {Outcome: "maybe"},
		"date":    {From: "01-02-2026"},
	} {
		if _, err := service.List(context.Background(), request); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("%s: expected invalid input, got %v", name, err)
		}
	}
}

func TestListFiltersByDateRange(t *testing.T) {
	service := newTestService()
	props, err := service.List(context.Background(), FilterRequest{
		From: "2026-02-01",
		To:   "2026-02-01",
	})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if props.Page.Total != 3 {
		t.Fatalf("expected same-day entries, got %#v", props)
	}
	props, err = service.List(context.Background(), FilterRequest{From: "2026-02-02"})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if props.Page.Total != 0 {
		t.Fatalf("expected empty future range, got %#v", props)
	}
}

func TestListDefaultsAndClampsPerPage(t *testing.T) {
	service := newTestService()

	props, err := service.List(context.Background(), FilterRequest{})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if props.Page.PerPage != pagination.DefaultPerPage {
		t.Fatalf("expected the package default, got %d", props.Page.PerPage)
	}

	props, err = service.List(context.Background(), FilterRequest{PerPage: 10})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if props.Page.PerPage != 10 {
		t.Fatalf("explicit per_page must win, got %d", props.Page.PerPage)
	}

	props, err = service.List(
		context.Background(),
		FilterRequest{PerPage: pagination.MaxPerPage + 1},
	)
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if props.Page.PerPage != pagination.MaxPerPage {
		t.Fatalf("per_page must clamp to the cap, got %d", props.Page.PerPage)
	}
}

func TestListEchoesRequestedDateBounds(t *testing.T) {
	service := newTestService()
	props, err := service.List(context.Background(), FilterRequest{
		From: "2026-02-01",
		To:   "2026-02-01",
	})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	// A date-only `to` is expanded to an exclusive end-of-day instant for the
	// query, but the page must keep showing the inclusive day the user picked.
	if props.Filters.To != "2026-02-01" || props.Filters.From != "2026-02-01" {
		t.Fatalf("expected requested bounds, got %#v", props.Filters)
	}

	props, err = service.List(context.Background(), FilterRequest{To: "2026-02-01T06:00:00Z"})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if props.Filters.To != "2026-02-01T06:00:00Z" {
		t.Fatalf("expected exact instant, got %q", props.Filters.To)
	}
}
