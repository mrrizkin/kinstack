// Package users owns user accounts and their roles.
package users

import (
	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/authz"
	platforminertia "kinstack/internal/platform/inertia"

	"github.com/gofiber/fiber/v3"
)

// Dependency contains the repository, actor boundary, Inertia adapter, and
// logger used by the feature.
type Dependency struct {
	Repository    Repository
	Authorization *authz.Service
	Inviter       Inviter
	Revoker       CredentialRevoker
	Actor         *actor.Actor
	Logger        applog.Log
	Inertia       *platforminertia.Adapter
}

// Feature owns the user account service and HTTP entry points.
type Feature struct {
	service *Service
	handler *Handler
}

// New constructs the users feature from explicit dependencies.
func New(d Dependency) *Feature {
	assert.Assert(d.Repository != nil, "users repository is required")
	assert.Assert(d.Actor != nil, "users actor boundary is required")
	assert.Assert(d.Logger != nil, "users logger is required")
	assert.Assert(d.Inertia != nil, "users inertia adapter is required")

	service := NewService(ServiceDependency{
		Repository:    d.Repository,
		Authorization: d.Authorization,
		Inviter:       d.Inviter,
		Revoker:       d.Revoker,
	})
	handler := NewHandler(HandlerDependency{
		Service: service,
		Actor:   d.Actor,
		Logger:  d.Logger,
		Inertia: d.Inertia,
	})
	return &Feature{service: service, handler: handler}
}

// Mount registers user management routes on the provided group.
func (f *Feature) Mount(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "users feature must be initialized")
	f.handler.Mount(router)
}

// Service exposes the feature service for composition-root adapters.
func (f *Feature) Service() *Service {
	assert.Assert(f != nil && f.service != nil, "users feature must be initialized")
	return f.service
}
