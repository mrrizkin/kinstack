package audit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	platformactor "kinstack/internal/platform/actor"
	"kinstack/internal/platform/session"

	"github.com/gofiber/fiber/v3"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func TestMemoryRepositoryFiltersAndPreservesAuditEvents(t *testing.T) {
	recorder := New(NewMemoryRepository())
	first := Event{
		OccurredAt: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC),
		RequestID:  "req-1",
		ActorKey:   "H1i2J3k4L5m6N7o8P9q0R",
		Action:     "http.mutate",
		Resource:   "/users/:publicID/active",
		Method:     http.MethodPost,
		Status:     http.StatusConflict,
		Outcome:    "failure",
		Metadata:   map[string]string{"source": "http"},
	}
	if err := recorder.Record(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	first.Metadata["source"] = "mutated-after-record"
	if err := recorder.Record(context.Background(), Event{
		OccurredAt: first.OccurredAt.Add(time.Minute),
		ActorKey:   "G1h2I3j4K5l6M7n8O9p0Q",
		Action:     "http.read",
		Resource:   "/users",
		Method:     http.MethodGet,
		Status:     http.StatusOK,
		Outcome:    "success",
	}); err != nil {
		t.Fatal(err)
	}

	items, err := recorder.List(
		context.Background(),
		ListQuery{ActorKey: "H1i2J3k4L5m6N7o8P9q0R", Limit: 10},
	)
	if err != nil || len(items) != 1 {
		t.Fatalf("owner audit events=%#v err=%v", items, err)
	}
	if items[0].ID != 1 || items[0].Metadata["source"] != "http" || items[0].Outcome != "failure" {
		t.Fatalf("audit event was not preserved: %#v", items[0])
	}
	if _, err := recorder.List(context.Background(), ListQuery{Limit: -1}); !errors.Is(
		err,
		ErrInvalidEvent,
	) {
		t.Fatalf("invalid list limit error=%v", err)
	}
	if err := recorder.Record(
		context.Background(),
		Event{
			Action:   "bad",
			Resource: "/bad",
			Method:   http.MethodGet,
			Status:   200,
			Outcome:  "unknown",
		},
	); !errors.Is(
		err,
		ErrInvalidEvent,
	) {
		t.Fatalf("invalid outcome error=%v", err)
	}
}

func TestMiddlewareRecordsProtectedReadsAndMutations(t *testing.T) {
	recorder := New(NewMemoryRepository())
	actorBoundary := platformactor.NewMemoryActor("H1i2J3k4L5m6N7o8P9q0R")
	store := session.NewMemoryStore(time.Minute)
	app := fiber.New()
	app.Use(session.NewMiddleware(store, time.Minute))
	app.Use(
		Middleware(
			recorder,
			actorBoundary,
			func(err error) { t.Errorf("record audit event: %v", err) },
		),
	)
	app.Get("/login", func(c fiber.Ctx) error {
		if err := actorBoundary.SetActor(
			c,
			platformactor.SessionActor{PublicID: "H1i2J3k4L5m6N7o8P9q0R"},
		); err != nil {
			return err
		}
		return c.SendStatus(http.StatusNoContent)
	})
	app.Get("/users", func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	app.Post("/e/:eventId/register", func(c fiber.Ctx) error {
		return fiber.NewError(http.StatusConflict, "registration changed")
	})
	app.Get("/public", func(c fiber.Ctx) error { return c.SendStatus(http.StatusOK) })

	login, err := app.Test(httptest.NewRequest(http.MethodGet, "/login", nil))
	if err != nil {
		t.Fatal(err)
	}
	cookies := ""
	for _, cookie := range login.Cookies() {
		if cookies != "" {
			cookies += "; "
		}
		cookies += cookie.Name + "=" + cookie.Value
	}
	request := httptest.NewRequest(http.MethodGet, "/users", nil)
	request.Header.Set("Cookie", cookies)
	request.Header.Set("X-Request-ID", "req-read")
	response, err := app.Test(request)
	if err != nil || response.StatusCode != http.StatusOK {
		t.Fatalf("protected read status=%d err=%v", response.StatusCode, err)
	}
	mutation := httptest.NewRequest(http.MethodPost, "/e/B1c2D3e4F5g6H7i8J9k0L/register", nil)
	mutation.Header.Set("Cookie", cookies)
	mutation.Header.Set("X-Request-ID", "req-mutate")
	response, err = app.Test(mutation)
	if err != nil || response.StatusCode != http.StatusConflict {
		t.Fatalf("failed mutation status=%d err=%v", response.StatusCode, err)
	}
	if _, err := app.Test(httptest.NewRequest(http.MethodGet, "/public", nil)); err != nil {
		t.Fatal(err)
	}

	items, err := recorder.List(
		context.Background(),
		ListQuery{ActorKey: "H1i2J3k4L5m6N7o8P9q0R", Limit: 10},
	)
	if err != nil || len(items) != 2 {
		t.Fatalf("actor audit events=%#v err=%v", items, err)
	}
	if items[0].Action != "http.mutate" || items[0].Status != http.StatusConflict ||
		items[0].Outcome != "failure" ||
		items[0].Resource != "/e/:eventId/register" {
		t.Fatalf("mutation audit=%#v", items[0])
	}
	if items[1].Action != "http.read" || items[1].Status != http.StatusOK ||
		items[1].RequestID != "req-read" {
		t.Fatalf("read audit=%#v", items[1])
	}
}

func TestSQLRecorderContract(t *testing.T) {
	dsn := os.Getenv("KINSTACK_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("PostgreSQL is unavailable: %v", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var table string
	if err := db.GetContext(ctx, &table, `SELECT to_regclass('audit_events')`); err != nil ||
		table == "" {
		t.Skip("audit events migration is not applied")
	}

	requestID := fmt.Sprintf("sql-audit-%d", time.Now().UnixNano())
	repository, err := NewSQLRepository(RepositoryDependency{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()
	recorder := New(repository)
	if err := recorder.Record(ctx, Event{
		OccurredAt: time.Now().UTC(),
		RequestID:  requestID,
		ActorKey:   "A3b4C5d6E7f8G9h0I1j2K",
		Action:     "http.mutate",
		Resource:   "/access/roles/:code",
		Method:     http.MethodPatch,
		Status:     http.StatusOK,
		Outcome:    "success",
		Metadata:   map[string]string{"source": "contract"},
	}); err != nil {
		t.Fatal(err)
	}
	items, err := recorder.List(
		ctx,
		ListQuery{ActorKey: "A3b4C5d6E7f8G9h0I1j2K", Action: "http.mutate", Limit: 10},
	)
	if err != nil || len(items) == 0 {
		t.Fatalf("SQL audit events=%#v err=%v", items, err)
	}
	found := false
	for _, item := range items {
		if item.RequestID == requestID {
			found = true
			if item.Metadata["source"] != "contract" || item.Resource != "/access/roles/:code" {
				t.Fatalf("unexpected SQL audit event=%#v", item)
			}
		}
	}
	if !found {
		t.Fatalf("SQL audit request %q not found in %#v", requestID, items)
	}
}

func TestMemoryRepositoryDeleteBeforePrunesOldEvents(t *testing.T) {
	repository := NewMemoryRepository()
	recorded := New(repository)
	ctx := context.Background()

	for index, occurredAt := range []time.Time{
		time.Now().UTC().Add(-48 * time.Hour),
		time.Now().UTC().Add(-30 * time.Minute),
	} {
		if _, err := repository.Insert(ctx, AuditRecord{
			OccurredAt: occurredAt,
			RequestID:  fmt.Sprintf("prune-%d", index),
			Action:     "http.mutate",
			Outcome:    "success",
		}); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	deleted, err := recorded.Prune(ctx, time.Now().UTC().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("deleted=%d, want 1", deleted)
	}
	remaining, err := recorded.List(ctx, ListQuery{Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(remaining) != 1 {
		t.Fatalf("remaining=%d, want 1", len(remaining))
	}
	if _, err := recorded.Prune(ctx, time.Time{}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("expected ErrInvalidEvent for a zero cutoff, got %v", err)
	}
}

func TestSQLRepositoryDeleteBeforePrunesOldRows(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		t.Skip("DATABASE_URL is not set")
	}
	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		t.Skipf("PostgreSQL is unavailable: %v", err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var table string
	if err := db.GetContext(
		ctx,
		&table,
		`SELECT to_regclass('audit_events')`,
	); err != nil ||
		table == "" {
		t.Skip("audit events migration is not applied")
	}

	repository, err := NewSQLRepository(RepositoryDependency{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()

	requestID := fmt.Sprintf("prune-audit-%d", time.Now().UnixNano())
	for _, occurredAt := range []time.Time{
		time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Now().UTC(),
	} {
		if _, err := repository.Insert(ctx, AuditRecord{
			OccurredAt: occurredAt,
			RequestID:  requestID,
			Action:     "http.mutate",
			Outcome:    "success",
		}); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM audit_events WHERE request_id = $1`, requestID)
	})

	deleted, err := repository.DeleteBefore(ctx, time.Now().UTC().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("prune: %v", err)
	}
	if deleted < 1 {
		t.Fatalf("expected at least one pruned row, got %d", deleted)
	}
	var remaining int
	if err := db.GetContext(ctx, &remaining,
		`SELECT count(*) FROM audit_events WHERE request_id = $1`, requestID,
	); err != nil {
		t.Fatalf("count remaining: %v", err)
	}
	if remaining != 1 {
		t.Fatalf("expected the recent row to survive, got %d rows", remaining)
	}
}
