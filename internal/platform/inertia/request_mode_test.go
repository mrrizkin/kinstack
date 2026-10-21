package inertia

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestWantsJSONDetectsMachineClients(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		if WantsJSON(c) {
			return c.SendString("json")
		}
		return c.SendString("page")
	})

	cases := map[string]string{
		"application/json":                "json",
		"application/json, */*":           "json",
		"text/html,application/xhtml+xml": "page",
		"":                                "page",
	}
	for accept, want := range cases {
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		if accept != "" {
			request.Header.Set("Accept", accept)
		}
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		body := make([]byte, 16)
		n, _ := response.Body.Read(body)
		response.Body.Close()
		if got := string(body[:n]); got != want {
			t.Errorf("accept %q -> %q, want %q", accept, got, want)
		}
	}
}

// Inertia visits must never be mistaken for machine clients, even when a proxy
// or a client overrides the Accept header.
func TestWantsJSONIgnoresInertiaVisits(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		if WantsJSON(c) {
			return c.SendString("json")
		}
		return c.SendString("page")
	})

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("X-Inertia", "true")
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer response.Body.Close()
	body := make([]byte, 16)
	n, _ := response.Body.Read(body)
	if got := string(body[:n]); got != "page" {
		t.Fatalf("inertia visit -> %q, want page", got)
	}
}
