package users

import (
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
			response, err := app.Test(httptest.NewRequest(http.MethodGet, target, nil))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != fiber.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusUnprocessableEntity)
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
			response, err := app.Test(httptest.NewRequest(http.MethodGet, target, nil))
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer response.Body.Close()
			if response.StatusCode != fiber.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusUnprocessableEntity)
			}
		})
	}
}
