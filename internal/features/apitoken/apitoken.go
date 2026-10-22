// Package apitoken owns personal API tokens: long-lived bearer credentials a
// signed-in user can create, list, and revoke from their account.
package apitoken

import (
	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/pkg/pagination"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/authz"
	platforminertia "kinstack/internal/platform/inertia"

	"github.com/gofiber/fiber/v3"
)

type Dependency struct {
	Repository    Repository
	Actor         *actor.Actor
	Authorization *authz.Service
	PageSize      pagination.PageSizeReader
	Logger        applog.Log
	Inertia       *platforminertia.Adapter
}

// Feature owns the API token service and HTTP entry points.
type Feature struct {
	service *Service
	handler *Handler
}

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
		PageSize:      d.PageSize,
		Logger:        d.Logger,
		Inertia:       d.Inertia,
	})
	return &Feature{service: service, handler: handler}
}

func (f *Feature) Mount(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "api token feature must be initialized")
	f.handler.Mount(router)
}

func (f *Feature) Service() *Service {
	assert.Assert(f != nil && f.service != nil, "api token feature must be initialized")
	return f.service
}
