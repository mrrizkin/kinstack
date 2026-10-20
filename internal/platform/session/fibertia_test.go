package session

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestInertiaSessionStoreFlashRoundTrip(t *testing.T) {
	fiberStore := NewMemoryStore(time.Minute)
	inertiaStore := NewInertiaSessionStore(fiberStore)
	app := fiber.New()
	app.Use(NewMiddleware(fiberStore, time.Minute))
	app.Post("/set", func(c fiber.Ctx) error {
		return inertiaStore.SetFlash(c, map[string]any{"error": "invalid credentials"})
	})
	app.Get("/read", func(c fiber.Ctx) error {
		flash, err := inertiaStore.GetFlash(c)
		if err != nil {
			return err
		}
		return c.JSON(flash)
	})

	setResponse, err := app.Test(httptest.NewRequest("POST", "/set", nil))
	if err != nil {
		t.Fatal(err)
	}
	cookie := sessionCookie(setResponse.Header.Get("Set-Cookie"))
	if cookie == "" {
		t.Fatal("expected session cookie after setting flash")
	}

	read := func(cookie string) (string, map[string]any) {
		t.Helper()
		request := httptest.NewRequest("GET", "/read", nil)
		request.Header.Set("Cookie", cookie)
		response, requestErr := app.Test(request)
		if requestErr != nil {
			t.Fatal(requestErr)
		}
		defer response.Body.Close()
		var payload map[string]any
		if decodeErr := json.NewDecoder(response.Body).Decode(&payload); decodeErr != nil {
			t.Fatal(decodeErr)
		}
		return sessionCookie(response.Header.Get("Set-Cookie")), payload
	}

	updatedCookie, first := read(cookie)
	if first["error"] != "invalid credentials" {
		t.Fatalf("flash=%#v", first)
	}
	if updatedCookie == "" {
		updatedCookie = cookie
	}
	_, second := read(updatedCookie)
	if len(second) != 0 {
		t.Fatalf("flash was not consumed: %#v", second)
	}
}

func sessionCookie(setCookie string) string {
	if setCookie == "" {
		return ""
	}
	return strings.Split(setCookie, ";")[0]
}
