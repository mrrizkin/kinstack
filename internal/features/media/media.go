package media

import (
	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/pkg/pagination"
	"kinstack/internal/platform/actor"
	platforminertia "kinstack/internal/platform/inertia"

	"github.com/gofiber/fiber/v3"
)

// Dependency contains the boundaries used by the media feature.
type Dependency struct {
	Repository Repository
	Storage    MediaStorage
	Actor      *actor.Actor
	PageSize   pagination.PageSizeReader
	Logger     applog.Log
	Inertia    *platforminertia.Adapter
}

// Feature owns the media service and HTTP entry point.
type Feature struct {
	service *Service
	handler *Handler
}

// New constructs the media feature from explicit dependencies.
func New(d Dependency) *Feature {
	assert.Assert(d.Repository != nil, "media repository is required")
	assert.Assert(d.Storage != nil, "media storage is required")
	assert.Assert(d.Actor != nil, "media actor is required")
	assert.Assert(d.Logger != nil, "media logger is required")
	assert.Assert(d.Inertia != nil, "media inertia adapter is required")

	service := NewService(ServiceDependency{Repository: d.Repository, Storage: d.Storage})
	handler := NewHandler(HandlerDependency{
		Service:  service,
		Actor:    d.Actor,
		PageSize: d.PageSize,
		Logger:   d.Logger,
		Inertia:  d.Inertia,
	})
	return &Feature{service: service, handler: handler}
}

// Mount registers media routes on the guarded workspace.
func (f *Feature) Mount(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "media feature must be initialized")
	f.handler.Mount(router)
}

// Service exposes the feature service for composition-root adapters.
func (f *Feature) Service() *Service {
	assert.Assert(f != nil && f.service != nil, "media feature must be initialized")
	return f.service
}
