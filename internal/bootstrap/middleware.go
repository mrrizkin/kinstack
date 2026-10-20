package bootstrap

import (
	"context"

	"kinstack/config"
	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/audit"
	"kinstack/internal/platform/authz"
	"kinstack/internal/platform/observability"
	"kinstack/internal/platform/pdf"
	"kinstack/internal/platform/sessionregistry"

	"codeberg.org/mrrizkin/fibertia"
	"github.com/gofiber/fiber/v3"
	fibersession "github.com/gofiber/fiber/v3/middleware/session"
)

// GlobalMiddlewareDependency contains the boundaries used by global middleware.
type GlobalMiddlewareDependency struct {
	App           *fiber.App
	Config        config.Config
	Inertia       *fibertia.Fibertia
	Session       fiber.Handler
	SessionStore  *fibersession.Store
	SharedStorage fiber.Storage
	Actor         *actor.Actor
	Authz         *authz.Service
	Audit         audit.Recorder
	Log           *log.Logger
	Documents     *pdf.Client
	Sessions      *sessionregistry.Service
	BearerAuth    func(context.Context, string) (*actor.SessionActor, error)
	Telemetry     *observability.Telemetry
}

// RegisterGlobalMiddleware wires security, request context, and Inertia
// middleware in the correct order.
func RegisterGlobalMiddleware(d GlobalMiddlewareDependency) {
	assert.Assert(d.App != nil, "app is required")
	assert.Assert(d.Inertia != nil, "inertia is required")
	assert.Assert(d.Session != nil, "session middleware is required")
	assert.Assert(d.SessionStore != nil, "session store is required")
	assert.Assert(d.Actor != nil, "actor is required")
	assert.Assert(d.Log != nil, "logger is required")
	assert.Assert(d.Documents != nil, "document renderer is required")
	assert.Assert(d.BearerAuth != nil, "bearer authenticator is required")

	registerSecurityMiddleware(d)
	registerContextMiddleware(d)

	d.Log.Infow("Middleware configured")
}
