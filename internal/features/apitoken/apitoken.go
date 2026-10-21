// Package apitoken owns personal API tokens: long-lived bearer credentials a
// signed-in user can create, list, and revoke from their account.
package apitoken

import (
	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/authz"
	platforminertia "kinstack/internal/platform/inertia"

	"github.com/gofiber/fiber/v3"
)

// Dependency contains the persistence boundary, actor boundary, authorization
// resolver, Inertia adapter, and logger used by the feature.
type Dependency struct {
	Repository    Repository
	Actor         *actor.Actor
	Authorization *authz.Service
	Logger        applog.Log
	Inertia       *platforminertia.Adapter
}

// Feature owns the API token service and HTTP entry points.
type Feature struct {
	service *Service
	handler *Handler
}

// New constructs the API token feature from explicit dependencies.
func New(d Dependency) *Feature {
	assert.Assert(d.Repository != nil, "api token repository is required")
	assert.Assert(d.Actor != nil, "api token actor boundary is required")
	assert.Assert(d.Logger != nil, "api token logger is required")
	assert.Assert(d.Inertia != nil, "api token inertia adapter is required")

	service := NewService(ServiceDependency{Repository: d.Repository})
	handler := NewHandler(HandlerDependency{
		Service:       service,
		Actor:         d.Actor,
		Authorization: d.Authorization,
		Logger:        d.Logger,
		Inertia:       d.Inertia,
	})
	return &Feature{service: service, handler: handler}
}

// Mount registers the personal API token routes on the authenticated workspace.
func (f *Feature) Mount(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "api token feature must be initialized")
	f.handler.Mount(router)
}

// Service exposes the feature service for composition-root adapters.
func (f *Feature) Service() *Service {
	assert.Assert(f != nil && f.service != nil, "api token feature must be initialized")
	return f.service
}
