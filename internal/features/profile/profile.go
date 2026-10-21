// Package profile owns the signed-in user's profile, preferences, and password.
package profile

import (
	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	platforminertia "kinstack/internal/platform/inertia"
	"kinstack/internal/platform/sessionregistry"
	"kinstack/internal/platform/twofactor"

	"github.com/gofiber/fiber/v3"
)

// Dependency contains the repository, actor boundary, session registry,
// two-factor service, Inertia adapter, and logger used by the feature.
type Dependency struct {
	Repository Repository
	Actor      *actor.Actor
	Logger     applog.Log
	Inertia    *platforminertia.Adapter
	Sessions   *sessionregistry.Service
	TwoFactor  *twofactor.Service
}

// Feature owns the profile service and HTTP entry points.
type Feature struct {
	service *Service
	handler *Handler
}

// New constructs the profile feature from explicit dependencies.
func New(d Dependency) *Feature {
	assert.Assert(d.Repository != nil, "profile repository is required")
	assert.Assert(d.Actor != nil, "profile actor boundary is required")
	assert.Assert(d.Logger != nil, "profile logger is required")
	assert.Assert(d.Inertia != nil, "profile inertia adapter is required")

	service := NewService(ServiceDependency{Repository: d.Repository})
	handler := NewHandler(HandlerDependency{
		Service:   service,
		Actor:     d.Actor,
		Logger:    d.Logger,
		Inertia:   d.Inertia,
		Sessions:  d.Sessions,
		TwoFactor: d.TwoFactor,
	})
	return &Feature{service: service, handler: handler}
}

// Mount registers profile routes on the authenticated workspace group.
func (f *Feature) Mount(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "profile feature must be initialized")
	f.handler.Mount(router)
}

// Service exposes the feature service for composition-root adapters.
func (f *Feature) Service() *Service {
	assert.Assert(f != nil && f.service != nil, "profile feature must be initialized")
	return f.service
}
