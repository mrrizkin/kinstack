package authz

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"kinstack/internal/platform/actor"

	"github.com/gofiber/fiber/v3"
)

func newGuardTestApp(t *testing.T, policy RoutePolicy) (*fiber.App, *Service) {
	t.Helper()
	repository := NewMemoryRepository()
	repository.SeedRole(Role{Code: "staff", Name: "Kasir"})
	repository.SeedRolePermissions("staff", PermissionUsersView)
	repository.SeedUser("staff-1", "staff")
	service := New(ServiceDependency{Repository: repository})

	boundary := actor.New(actor.Dependency{Repository: actor.NewMemoryRepository("staff-1")})
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("actor", &actor.SessionActor{PublicID: "staff-1", Role: "staff"})
		return c.Next()
	})
	guard := NewGuardRouter(app.Group("/w"), boundary, service, policy)
	guard.Get("/allowed", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	guard.Get("/denied", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	return app, service
}

func TestGuardRouterAllowsGrantedPermission(t *testing.T) {
	app, _ := newGuardTestApp(t, RoutePolicy{
		"GET /allowed": PermissionUsersView,
		"GET /denied":  PermissionUsersManage,
	})

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/w/allowed", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
}

func TestGuardRouterForbidsMissingPermission(t *testing.T) {
	app, _ := newGuardTestApp(t, RoutePolicy{
		"GET /allowed": PermissionUsersView,
		"GET /denied":  PermissionUsersManage,
	})

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/w/denied", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403, got %d", response.StatusCode)
	}
}

func TestGuardRouterFailsClosedWithoutPolicy(t *testing.T) {
	repository := NewMemoryRepository()
	service := New(ServiceDependency{Repository: repository})
	boundary := actor.New(actor.Dependency{Repository: actor.NewMemoryRepository("staff-1")})
	app := fiber.New()
	guard := NewGuardRouter(app, boundary, service, RoutePolicy{})

	defer func() {
		if recover() == nil {
			t.Fatalf("expected missing route policy to panic")
		}
	}()
	guard.Get("/unlisted", func(c fiber.Ctx) error { return nil })
}

func TestGuardRouterRejectsAnonymous(t *testing.T) {
	repository := NewMemoryRepository()
	repository.SeedRole(Role{Code: "staff", Name: "Kasir"})
	repository.SeedRolePermissions("staff", PermissionUsersView)
	service := New(ServiceDependency{Repository: repository})
	boundary := actor.New(actor.Dependency{Repository: actor.NewMemoryRepository("staff-1")})

	app := fiber.New()
	guard := NewGuardRouter(
		app,
		boundary,
		service,
		RoutePolicy{"GET /allowed": PermissionUsersView},
	)
	guard.Get("/allowed", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/allowed", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if response.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", response.StatusCode)
	}
}

func TestGuardRouterAppliesResourcePolicy(t *testing.T) {
	repository := NewMemoryRepository()
	repository.SeedRole(Role{Code: "staff", Name: "Kasir"})
	repository.SeedRolePermissions("staff", PermissionUsersManage)
	repository.SeedUser("staff-1", "staff")
	service := New(ServiceDependency{
		Repository: repository,
		Engine: NewEngine(OwnershipPolicy{
			PolicyName: "test.ownership",
			Permission: PermissionUsersManage,
			ExemptRole: adminRole,
			Message:    "hanya boleh mengubah data sendiri",
		}),
	})
	boundary := actor.New(actor.Dependency{Repository: actor.NewMemoryRepository("staff-1")})

	owner := "staff-1"
	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("actor", &actor.SessionActor{PublicID: "staff-1", Role: "staff"})
		return c.Next()
	})
	guard := NewGuardRouter(app, boundary, service, RoutePolicy{
		"POST /users/:publicID": PermissionUsersManage,
	}).WithResolvers(map[string]ResourceResolver{
		"POST /users/:publicID": func(fiber.Ctx) Resource {
			return Resource{ResourceOwner: owner}
		},
	})
	guard.Post("/users/:publicID", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })

	request := func() int {
		response, err := app.Test(httptest.NewRequest(http.MethodPost, "/users/user-1", nil))
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		return response.StatusCode
	}

	if code := request(); code != fiber.StatusOK {
		t.Fatalf("own record should be allowed, got %d", code)
	}
	owner = "staff-2"
	if code := request(); code != fiber.StatusForbidden {
		t.Fatalf("another user's record should be forbidden, got %d", code)
	}
}

func newBrowserSessionTestApp(t *testing.T, scopes []string) *fiber.App {
	t.Helper()
	repository := NewMemoryRepository()
	service := New(ServiceDependency{Repository: repository})
	boundary := actor.New(actor.Dependency{Repository: actor.NewMemoryRepository("staff-1")})

	app := fiber.New()
	app.Use(func(c fiber.Ctx) error {
		c.Locals("actor", &actor.SessionActor{
			PublicID: "staff-1",
			Role:     "staff",
			Scopes:   scopes,
		})
		return c.Next()
	})
	guard := NewGuardRouter(app, boundary, service, RoutePolicy{
		"POST /profile/password": BrowserSession,
	})
	guard.Post("/profile/password", func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusOK) })
	return app
}

func TestGuardRouterAllowsBrowserSession(t *testing.T) {
	app := newBrowserSessionTestApp(t, nil)

	response, err := app.Test(httptest.NewRequest(http.MethodPost, "/profile/password", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200 for a browser session, got %d", response.StatusCode)
	}
}

func TestGuardRouterRejectsTokenOnBrowserSessionRoute(t *testing.T) {
	for name, scopes := range map[string][]string{
		"no scopes":    {},
		"valid scopes": {"users.view"},
	} {
		t.Run(name, func(t *testing.T) {
			app := newBrowserSessionTestApp(t, scopes)

			response, err := app.Test(
				httptest.NewRequest(http.MethodPost, "/profile/password", nil),
			)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			if response.StatusCode != fiber.StatusForbidden {
				t.Fatalf("expected 403 for an API token, got %d", response.StatusCode)
			}
		})
	}
}

func TestBrowserSessionIsNotAssignable(t *testing.T) {
	if IsKnown(BrowserSession) {
		t.Fatal("BrowserSession is an internal sentinel and must not be assignable")
	}
	if IsKnown(Authenticated) {
		t.Fatal("Authenticated is an internal sentinel and must not be assignable")
	}
}
