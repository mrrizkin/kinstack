package actor

import (
	"context"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v3"
)

type countingRepository struct {
	calls atomic.Int32
}

func (r *countingRepository) FindByPublicID(context.Context, string) error {
	r.calls.Add(1)
	return nil
}

func newValidationApp(t *testing.T, boundary *Actor, mark bool) *fiber.App {
	t.Helper()
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		boundary.SetRequestActor(c, SessionActor{PublicID: "user-1"})
		if mark {
			if err := boundary.ValidateSession(c.Context(), boundary.ActorFromContext(c)); err != nil {
				return err
			}
			boundary.MarkValidated(c)
		}
		return c.Next()
	})
	app.Get("/", boundary.RequireKnownActor(), func(c fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})
	return app
}

func testStatus(t *testing.T, app *fiber.App, target string) int {
	t.Helper()
	resp, err := app.Test(httptest.NewRequest("GET", target, nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return resp.StatusCode
}

func TestRequireKnownActorReusesRequestValidation(t *testing.T) {
	repository := &countingRepository{}
	boundary := New(Dependency{Repository: repository})
	app := newValidationApp(t, boundary, true)

	if status := testStatus(t, app, "/"); status != fiber.StatusNoContent {
		t.Fatalf("status=%d, want %d", status, fiber.StatusNoContent)
	}
	if got := repository.calls.Load(); got != 1 {
		t.Fatalf("actor must be validated once per request, got %d lookups", got)
	}
}

func TestRequireKnownActorValidatesWithoutMarker(t *testing.T) {
	repository := &countingRepository{}
	boundary := New(Dependency{Repository: repository})
	app := newValidationApp(t, boundary, false)

	if status := testStatus(t, app, "/"); status != fiber.StatusNoContent {
		t.Fatalf("status=%d, want %d", status, fiber.StatusNoContent)
	}
	if got := repository.calls.Load(); got != 1 {
		t.Fatalf("expected one validation lookup, got %d", got)
	}
}
