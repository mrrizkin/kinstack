package bootstrap

import (
	"context"
	"net/http"
	"time"

	"kinstack/config"
	"kinstack/internal/features/apitoken"
	"kinstack/internal/features/audit"
	"kinstack/internal/features/auth"
	"kinstack/internal/features/media"
	"kinstack/internal/features/notifications"
	"kinstack/internal/features/passkeys"
	"kinstack/internal/features/profile"
	"kinstack/internal/features/system"
	"kinstack/internal/features/users"
	"kinstack/internal/pkg/assert"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/authz"

	"codeberg.org/mrrizkin/fibertia"
	"github.com/gofiber/fiber/v3"
)

// DependencyCheck is a bounded readiness probe for one runtime dependency.
type DependencyCheck func(context.Context) error

// ReadinessChecks contains infrastructure probes. Nil checks are reported as
// unavailable instead of allowing a partially wired process to receive traffic.
type ReadinessChecks struct {
	Database DependencyCheck
	Session  DependencyCheck
}

// WebRoutesDependency contains the routes and feature entry points to register.
type WebRoutesDependency struct {
	App           *fiber.App
	Config        config.Config
	Inertia       *fibertia.Fibertia
	Actor         *actor.Actor
	Authz         *authz.Service
	Auth          *auth.Feature
	Users         *users.Feature
	Profile       *profile.Feature
	APITokens     *apitoken.Feature
	Passkeys      *passkeys.Feature
	Audit         *audit.Feature
	Media         *media.Feature
	Notifications *notifications.Feature
	System        *system.Feature
	Readiness     ReadinessChecks
}

// RegisterWebRoutes wires the HTTP route table.
func RegisterWebRoutes(d WebRoutesDependency) {
	assert.Assert(d.App != nil, "app is required")
	assert.Assert(d.Config.App.Name != "", "app name is required")
	assert.Assert(d.Inertia != nil, "inertia adapter is required")
	assert.Assert(d.Actor != nil, "actor is required")
	assert.Assert(d.Authz != nil, "authorization service is required")
	assert.Assert(d.Auth != nil, "auth feature is required")
	assert.Assert(d.Users != nil, "users feature is required")
	assert.Assert(d.Profile != nil, "profile feature is required")
	assert.Assert(d.APITokens != nil, "api token feature is required")
	assert.Assert(d.Passkeys != nil, "passkeys feature is required")
	assert.Assert(d.Audit != nil, "audit feature is required")
	assert.Assert(d.System != nil, "system feature is required")

	d.App.Get("/health", healthHandler(d.Config))
	d.App.Get("/ready", readinessHandler(d.Config, d.Readiness))
	d.App.Get("/", func(c fiber.Ctx) error {
		if session := d.Actor.ActorFromContext(c); session != nil {
			return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard")
		}
		if err := c.Render("welcome/WelcomeView", fiber.Map{}); err != nil {
			return err
		}
		return nil
	})

	d.Auth.Mount(d.App)
	d.System.Mount(d.App)
	if d.Config.Auth.Passkey.Enabled {
		d.Passkeys.Mount(d.App)
	}

	// Workspace routes are mounted at the root; there is no landing page in the
	// base, so the root handler dispatches each actor to its home surface.
	knownActor := d.Actor.RequireKnownActor()
	guard := authz.NewGuardRouter(d.App, d.Actor, d.Authz, routePolicy).
		WithMiddleware(knownActor)

	guard.Get("/dashboard", func(c fiber.Ctx) error {
		return c.Render("dashboard/IndexView", fiber.Map{"metrics": dashboardMetrics(c, d)})
	})

	d.Profile.Mount(guard)
	if d.Config.Auth.APIToken.Enabled {
		d.APITokens.Mount(guard)
	}
	if d.Config.Auth.Passkey.Enabled {
		d.Passkeys.MountWorkspace(guard)
	}
	d.Users.Mount(guard)
	d.Audit.Mount(guard)
	if d.Media != nil {
		d.Media.Mount(guard)
	}
	d.Notifications.Mount(guard)
}

// dashboardMetrics builds the small workspace summary shown on the dashboard.
// Counts are best-effort: a failed query renders a zero rather than failing the
// whole page, because the dashboard remains useful without its cards.
func dashboardMetrics(c fiber.Ctx, d WebRoutesDependency) fiber.Map {
	metrics := fiber.Map{
		"users_total": 0, "users_active": 0, "roles_total": 0,
		"audit_24h": 0, "media_total": 0, "notifications_unread": 0,
	}
	if users, err := d.Users.Service().List(c.Context()); err == nil {
		active := 0
		for _, user := range users {
			if user.IsActive {
				active++
			}
		}
		metrics["users_total"] = len(users)
		metrics["users_active"] = active
	}
	if roles, err := d.Authz.ListRoles(c.Context()); err == nil {
		metrics["roles_total"] = len(roles)
	}
	from := time.Now().Add(-24 * time.Hour).UTC().Format(time.RFC3339)
	if page, err := d.Audit.Service().List(c.Context(), audit.FilterRequest{From: from, Page: 1, PerPage: 1}); err == nil {
		metrics["audit_24h"] = page.Total
	}
	session := d.Actor.ActorFromContext(c)
	if session != nil {
		if d.Media != nil {
			if assets, err := d.Media.Service().List(c.Context(), session.PublicID); err == nil {
				metrics["media_total"] = len(assets)
			}
		}
		if page, err := d.Notifications.Service().List(c.Context(), session.PublicID); err == nil {
			metrics["notifications_unread"] = page.Unread
		}
	}
	return metrics
}

// routePolicy is the single route -> permission table for the authenticated
// workspace. GuardRouter fails closed: a route registered without an entry here
// panics at wiring time. Auth and system surfaces are mounted outside it.
var routePolicy = authz.RoutePolicy{
	"GET /dashboard": authz.Authenticated,

	"GET /profile":                            authz.Authenticated,
	"POST /profile":                           authz.Authenticated,
	"POST /profile/preferences":               authz.Authenticated,
	"POST /profile/password":                  authz.Authenticated,
	"POST /profile/sessions/:publicID/revoke": authz.Authenticated,
	"POST /profile/sessions/revoke-others":    authz.Authenticated,
	"POST /profile/two-factor/begin":          authz.Authenticated,
	"POST /profile/two-factor/confirm":        authz.Authenticated,
	"POST /profile/two-factor/disable":        authz.Authenticated,
	"POST /profile/two-factor/recovery-codes": authz.Authenticated,

	"GET /api-tokens":                   authz.Authenticated,
	"POST /api-tokens":                  authz.Authenticated,
	"POST /api-tokens/:publicID/revoke": authz.Authenticated,

	"GET /passkeys":                       authz.Authenticated,
	"POST /passkeys/register/options":     authz.Authenticated,
	"POST /passkeys/register":             authz.Authenticated,
	"POST /passkeys/:credentialID/delete": authz.Authenticated,

	"GET /users": authz.PermissionUsersView, "POST /users": authz.PermissionUsersManage,
	"POST /users/invite":             authz.PermissionUsersManage,
	"PUT /users/:publicID":           authz.PermissionUsersManage,
	"POST /users/:publicID/password": authz.PermissionUsersManage,
	"POST /users/:publicID/active":   authz.PermissionUsersManage,

	"GET /access":                                     authz.PermissionAccessManage,
	"POST /access/roles":                              authz.PermissionAccessManage,
	"PUT /access/roles/:code":                         authz.PermissionAccessManage,
	"POST /access/roles/:code/delete":                 authz.PermissionAccessManage,
	"POST /access/roles/:code/permissions":            authz.PermissionAccessManage,
	"POST /access/users/:publicID/role":               authz.PermissionAccessManage,
	"POST /access/users/:publicID/permissions":        authz.PermissionAccessManage,
	"POST /access/users/:publicID/permissions/delete": authz.PermissionAccessManage,

	"GET /audit": authz.PermissionAuditView,

	"GET /notifications":                 authz.Authenticated,
	"GET /notifications/feed":            authz.Authenticated,
	"POST /notifications/read-all":       authz.Authenticated,
	"POST /notifications/:publicID/read": authz.Authenticated,

	"GET /media":                   authz.PermissionMediaView,
	"GET /media/:publicID/file":    authz.PermissionMediaView,
	"POST /media":                  authz.PermissionMediaManage,
	"POST /media/:publicID/delete": authz.PermissionMediaManage,
}

func healthHandler(cfg config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"app":    cfg.App.Name,
			"env":    cfg.App.Env,
		})
	}
}

func readinessHandler(cfg config.Config, checks ReadinessChecks) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 3*time.Second)
		defer cancel()

		dependencies := fiber.Map{}
		ready := true
		check := func(name string, probe DependencyCheck) {
			if probe == nil {
				dependencies[name] = "unavailable"
				ready = false
				return
			}
			if err := probe(ctx); err != nil {
				dependencies[name] = "unavailable"
				ready = false
				return
			}
			dependencies[name] = "ok"
		}
		check("database", checks.Database)
		check("session", checks.Session)

		status := fiber.StatusOK
		state := "ready"
		if !ready {
			status = http.StatusServiceUnavailable
			state = "not_ready"
		}
		return c.Status(status).JSON(fiber.Map{
			"status":       state,
			"app":          cfg.App.Name,
			"dependencies": dependencies,
		})
	}
}
