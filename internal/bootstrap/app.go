package bootstrap

import (
	"kinstack/config"
	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/audit"
	"kinstack/internal/platform/database"
	"kinstack/internal/platform/email"
	"kinstack/internal/platform/inertia"
	"kinstack/internal/platform/observability"
	"kinstack/internal/platform/pdf"

	"github.com/gofiber/fiber/v3"
	fibersession "github.com/gofiber/fiber/v3/middleware/session"
)

type AppDependency struct {
	Config        config.Config
	Database      *database.DB
	Log           *log.Logger
	Session       fiber.Handler
	SessionStore  *fibersession.Store
	SharedStorage fiber.Storage
	Actor         *actor.Actor
	Audit         audit.Recorder
	Email         email.Sender
	Readiness     ReadinessChecks
	Telemetry     *observability.Telemetry
}

// NewApp composes the platform boundaries and feature slices into an HTTP app.
// The returned cleanup releases repositories the application owns.
func NewApp(d AppDependency) (*fiber.App, func()) {
	assert.Assert(d.Database != nil, "database is required")
	assert.Assert(d.Log != nil, "logger is required")
	assert.Assert(d.Session != nil, "session middleware is required")
	assert.Assert(d.SessionStore != nil, "session store is required")
	assert.Assert(d.Actor != nil, "actor boundary is required")
	assert.Assert(d.Audit != nil, "audit recorder is required")
	assert.Assert(d.Email != nil, "email sender is required")

	inertiaAdapter := inertia.New(inertia.Dependency{
		Environment:  d.Config.App.Env,
		SessionStore: d.SessionStore,
	})
	documentRenderer := pdf.New(pdf.Config{
		URL:     d.Config.Document.GotenbergURL,
		Timeout: d.Config.Document.Timeout,
	})
	app := newFiberApp(FiberAppDependency{
		Config:    d.Config,
		Inertia:   inertiaAdapter,
		Log:       d.Log,
		Telemetry: d.Telemetry,
	})

	deps := newAppDependencies(d, inertiaAdapter)

	middleware := GlobalMiddlewareDependency{
		App:           app,
		Config:        d.Config,
		Inertia:       inertiaAdapter,
		Session:       d.Session,
		SessionStore:  d.SessionStore,
		SharedStorage: d.SharedStorage,
		Actor:         d.Actor,
		Authz:         deps.Authorization,
		Audit:         d.Audit,
		Log:           d.Log,
		Documents:     documentRenderer,
		Sessions:      deps.SessionRegistry,
		BearerAuth:    deps.BearerAuth,
		Telemetry:     d.Telemetry,
	}

	// Transport middleware (headers, request ids, logging) must run first so
	// assets also carry CSP/HSTS; assets are mounted before the session and
	// authorization layers so cached files never touch the session store.
	registerTransportMiddleware(middleware)
	registerStaticAssets(app)
	RegisterGlobalMiddleware(middleware)

	RegisterWebRoutes(WebRoutesDependency{
		App:           app,
		Config:        d.Config,
		Inertia:       inertiaAdapter,
		Actor:         d.Actor,
		Authz:         deps.Authorization,
		Auth:          deps.Auth,
		Users:         deps.Users,
		Profile:       deps.Profile,
		APITokens:     deps.APITokens,
		Passkeys:      deps.Passkeys,
		Audit:         deps.Audit,
		Media:         deps.Media,
		Notifications: deps.Notifications,
		System:        deps.System,
		Readiness:     d.Readiness,
	})

	return app, deps.close
}
