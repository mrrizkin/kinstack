// Package system owns explicit public service-state and help surfaces.
package system

import (
	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"

	"github.com/gofiber/fiber/v3"
)

type Dependency struct {
	Repository Repository
	Logger     applog.Log
}

type Feature struct {
	service *Service
	handler *Handler
}

// New constructs the system feature from explicit production dependencies.
func New(d Dependency) *Feature {
	assert.Assert(d.Repository != nil, "system repository is required")
	assert.Assert(d.Logger != nil, "system logger is required")

	service := NewService(ServiceDependency{
		Repository: d.Repository,
	})
	handler := NewHandler(HandlerDependency{
		Service: service,
		Logger:  d.Logger,
	})
	return &Feature{service: service, handler: handler}
}

// Mount registers the existing public system routes.
func (f *Feature) Mount(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "system feature must be initialized")
	f.handler.Mount(router)
}
