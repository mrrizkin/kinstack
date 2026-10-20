package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	platformactor "kinstack/internal/platform/actor"

	"github.com/gofiber/fiber/v3"
)

func TestRequireKnownActorHTTPBoundary(t *testing.T) {
	actorBoundary := platformactor.NewMemoryActor("A1b2C3d4E5f6G7h8I9j0K")
	app := fiber.New()
	app.Use(NewMiddleware(NewMemoryStore(time.Minute), time.Minute))
	app.Get("/sign-in", func(c fiber.Ctx) error {
		if err := actorBoundary.SetActor(
			c,
			platformactor.SessionActor{PublicID: c.Query("actor")},
		); err != nil {
			return err
		}
		return c.SendStatus(http.StatusNoContent)
	})
	app.Get(
		"/protected",
		actorBoundary.RequireKnownActor(),
		func(c fiber.Ctx) error {
			return c.SendStatus(http.StatusOK)
		},
	)

	guestResponse, err := app.Test(httptest.NewRequest(http.MethodGet, "/protected", nil))
	if err != nil {
		t.Fatalf("guest request: %v", err)
	}
	if guestResponse.StatusCode != http.StatusSeeOther {
		t.Fatalf("expected guest redirect, got %d", guestResponse.StatusCode)
	}

	loginResponse, err := app.Test(
		httptest.NewRequest(http.MethodGet, "/sign-in?actor=A1b2C3d4E5f6G7h8I9j0K", nil),
	)
	if err != nil {
		t.Fatalf("known actor request: %v", err)
	}
	cookies := loginResponse.Cookies()
	knownRequest := httptest.NewRequest(http.MethodGet, "/protected", nil)
	knownRequest.Header.Set("Cookie", cookiesHeader(cookies))
	knownResponse, err := app.Test(knownRequest)
	if err != nil {
		t.Fatalf("known actor protected request: %v", err)
	}
	if knownResponse.StatusCode != http.StatusOK {
		t.Fatalf("expected known actor access, got %d", knownResponse.StatusCode)
	}

	unknownLoginResponse, err := app.Test(
		httptest.NewRequest(http.MethodGet, "/sign-in?actor=usr_missing", nil),
	)
	if err != nil {
		t.Fatalf("unknown actor request: %v", err)
	}
	unknownRequest := httptest.NewRequest(http.MethodGet, "/protected", nil)
	unknownRequest.Header.Set("Cookie", cookiesHeader(unknownLoginResponse.Cookies()))
	unknownResponse, err := app.Test(unknownRequest)
	if err != nil {
		t.Fatalf("unknown actor protected request: %v", err)
	}
	if unknownResponse.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected unknown actor to be rejected, got %d", unknownResponse.StatusCode)
	}
}

func TestTwoFactorChallengeIsPreAuthAndRecoveryCodesAreOneTime(t *testing.T) {
	app := fiber.New()
	app.Use(NewMiddleware(NewMemoryStore(time.Minute), time.Minute))
	actorBoundary := platformactor.NewMemoryActor("A1b2C3d4E5f6G7h8I9j0K")
	app.Get("/challenge", func(c fiber.Ctx) error {
		if err := SetTwoFactorChallenge(c, "A1b2C3d4E5f6G7h8I9j0K"); err != nil {
			return err
		}
		if actorBoundary.ActorFromContext(c) != nil {
			t.Fatal("two-factor challenge must not establish an actor")
		}
		if actor, ok := PendingTwoFactorActor(c); !ok || actor != "A1b2C3d4E5f6G7h8I9j0K" {
			t.Fatalf("unexpected pending actor: %q %t", actor, ok)
		}
		if err := SetRecoveryCodes(c, []string{"ABCD-1234"}); err != nil {
			return err
		}
		codes, ok := ConsumeRecoveryCodes(c)
		if !ok || len(codes) != 1 || codes[0] != "ABCD-1234" {
			t.Fatalf("unexpected recovery codes: %#v %t", codes, ok)
		}
		if _, ok := ConsumeRecoveryCodes(c); ok {
			t.Fatal("recovery codes must be one-time")
		}
		return c.SendStatus(http.StatusNoContent)
	})
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/challenge", nil))
	if err != nil || response.StatusCode != http.StatusNoContent {
		t.Fatalf("challenge session: status=%d err=%v", response.StatusCode, err)
	}
}

func cookiesHeader(cookies []*http.Cookie) string {
	header := ""
	for index, cookie := range cookies {
		if index > 0 {
			header += "; "
		}
		header += cookie.Name + "=" + cookie.Value
	}
	return header
}
