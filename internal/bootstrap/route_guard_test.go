package bootstrap

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/authz"

	"github.com/gofiber/fiber/v3"
)

type knownActorRepository struct{}

func (knownActorRepository) FindByPublicID(context.Context, string) error {
	return nil
}

type emptyGrantRepository struct{ authz.Repository }

func (emptyGrantRepository) RolePermissions(context.Context, string) ([]authz.Permission, error) {
	return nil, nil
}

func (emptyGrantRepository) UserOverrides(context.Context, string) ([]authz.Override, error) {
	return nil, nil
}

// The guard router decides on the actor already attached to the request, so the
// test supplies one through a header instead of a session cookie.
func newGuardedApp(t *testing.T) *fiber.App {
	t.Helper()

	app := fiber.New()
	actorBoundary := actor.New(actor.Dependency{Repository: knownActorRepository{}})
	guard := authz.NewGuardRouter(app, actorBoundary, authz.New(authz.ServiceDependency{
		Repository: emptyGrantRepository{},
	}), authz.RoutePolicy{
		"GET /workspace":             authz.Authenticated,
		"POST /workspace/role":       authz.PermissionUsersView,
		"POST /workspace/credential": authz.BrowserSession,
	}).WithMiddleware(actorBoundary.RequireKnownActor())

	app.Use(func(c fiber.Ctx) error {
		if publicID := c.Get("X-Test-Actor"); publicID != "" {
			session := actor.SessionActor{PublicID: publicID, Role: "member"}
			if c.Get("X-Test-Token") != "" {
				session.Scopes = []string{"api_tokens"}
			}
			if c.Get("X-Test-Superuser") != "" {
				session.Superuser = true
			}
			actorBoundary.SetRequestActor(c, session)
		}
		return c.Next()
	})

	ok := func(c fiber.Ctx) error { return c.SendString("ok") }
	guard.Get("/workspace", ok)
	guard.Post("/workspace/role", ok)
	guard.Post("/workspace/credential", ok)
	return app
}

func guardedRequest(
	t *testing.T,
	app *fiber.App,
	method, path string,
	headers map[string]string,
) int {
	t.Helper()

	request := httptest.NewRequest(method, path, nil)
	request.Host = "localhost:3000"
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := app.Test(request, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response.StatusCode
}

func TestGuardedRoutesSendAnonymousRequestsToLogin(t *testing.T) {
	app := newGuardedApp(t)

	cases := map[string]struct{ method, path string }{
		"page":            {method: fiber.MethodGet, path: "/workspace"},
		"permission":      {method: fiber.MethodPost, path: "/workspace/role"},
		"browser session": {method: fiber.MethodPost, path: "/workspace/credential"},
	}
	for name, test := range cases {
		want := fiber.StatusUnauthorized
		if test.method == fiber.MethodGet {
			want = fiber.StatusSeeOther
		}
		if status := guardedRequest(t, app, test.method, test.path, nil); status != want {
			t.Fatalf("%s: status = %d, want %d", name, status, want)
		}
	}
}

func TestGuardedRoutesDistinguishTokensFromBrowserSessions(t *testing.T) {
	app := newGuardedApp(t)
	browser := map[string]string{"X-Test-Actor": "user-1"}
	token := map[string]string{"X-Test-Actor": "user-1", "X-Test-Token": "1"}

	if status := guardedRequest(
		t,
		app,
		fiber.MethodGet,
		"/workspace",
		token,
	); status != fiber.StatusOK {
		t.Fatalf("personal token on an authenticated route: status = %d", status)
	}
	if status := guardedRequest(
		t,
		app,
		fiber.MethodPost,
		"/workspace/credential",
		browser,
	); status != fiber.StatusOK {
		t.Fatalf("browser session on a credential route: status = %d", status)
	}
	if status := guardedRequest(
		t,
		app,
		fiber.MethodPost,
		"/workspace/credential",
		token,
	); status != fiber.StatusForbidden {
		t.Fatalf("personal token on a credential route: status = %d", status)
	}
	if status := guardedRequest(
		t,
		app,
		fiber.MethodPost,
		"/workspace/role",
		browser,
	); status != fiber.StatusForbidden {
		t.Fatalf("missing permission: status = %d", status)
	}
}

func TestGuardedRouteAllowsSuperuser(t *testing.T) {
	app := newGuardedApp(t)
	superuser := map[string]string{"X-Test-Actor": "root", "X-Test-Superuser": "1"}

	if status := guardedRequest(
		t,
		app,
		fiber.MethodPost,
		"/workspace/role",
		superuser,
	); status != fiber.StatusOK {
		t.Fatalf("superuser: status = %d", status)
	}
}
