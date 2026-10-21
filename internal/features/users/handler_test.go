package users

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
)

// Invalid pagination is rejected before the service is touched, so the handler
// can be exercised without wiring a full dependency graph.
func TestIndexRejectsInvalidPagination(t *testing.T) {
	handler := &Handler{}
	app := fiber.New()
	app.Get("/users", handler.index)

	cases := []string{
		"/users?page=0",
		"/users?page=abc",
		"/users?per_page=0",
		"/users?per_page=101",
		"/users?per_page=abc",
	}
	for _, target := range cases {
		t.Run(target, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, target, nil)
			request.Header.Set("Accept", "application/json")
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != fiber.StatusUnprocessableEntity {
				t.Fatalf(
					"status = %d, want %d",
					response.StatusCode,
					fiber.StatusUnprocessableEntity,
				)
			}
		})
	}
}

// The access page paginates its user table, so it must reject the same invalid
// page requests before touching the service.
func TestAccessRejectsInvalidPagination(t *testing.T) {
	handler := &Handler{}
	app := fiber.New()
	app.Get("/access", handler.access)

	for _, target := range []string{"/access?page=0", "/access?per_page=101", "/access?per_page=abc"} {
		t.Run(target, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, target, nil)
			request.Header.Set("Accept", "application/json")
			response, err := app.Test(request)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != fiber.StatusUnprocessableEntity {
				t.Fatalf(
					"status = %d, want %d",
					response.StatusCode,
					fiber.StatusUnprocessableEntity,
				)
			}
		})
	}
}

func TestWantsJSONDetectsMachineClients(t *testing.T) {
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		if wantsJSON(c) {
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
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if string(body) != want {
			t.Errorf("accept %q -> %q, want %q", accept, body, want)
		}
	}
}

func TestAccessPathForPreservesValidPage(t *testing.T) {
	handler := &Handler{}
	app := fiber.New()
	app.Get("/access", func(c fiber.Ctx) error {
		return c.SendString(handler.accessPathFor(c))
	})

	cases := map[string]string{
		"/access":               "/access",
		"/access?page=3":        "/access?page=3",
		"/access?page=abc":      "/access",
		"/access?page=0":        "/access",
		"/access?page=-2":       "/access",
		"/access?page=99999999": "/access",
	}
	for target, want := range cases {
		response, err := app.Test(httptest.NewRequest(http.MethodGet, target, nil))
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		body, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if string(body) != want {
			t.Errorf("%s -> %q, want %q", target, body, want)
		}
	}
}
