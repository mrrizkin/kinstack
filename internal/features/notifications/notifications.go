package notifications

import (
	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/pkg/pagination"
	"kinstack/internal/platform/actor"
	platforminertia "kinstack/internal/platform/inertia"

	"github.com/gofiber/fiber/v3"
)

type Dependency struct {
	Repository Repository
	Actor      *actor.Actor
	PageSize   pagination.PageSizeReader
	Logger     applog.Log
	Inertia    *platforminertia.Adapter
}

// Feature owns the notification service and HTTP entry point.
type Feature struct {
	service *Service
	handler *Handler
}

func New(d Dependency) *Feature {
	assert.Assert(d.Repository != nil, "notification repository is required")
	assert.Assert(d.Actor != nil, "notification actor is required")
	assert.Assert(d.Logger != nil, "notification logger is required")
	assert.Assert(d.Inertia != nil, "notification inertia adapter is required")

	service := NewService(ServiceDependency{Repository: d.Repository})
	handler := NewHandler(HandlerDependency{
		Service:  service,
		Actor:    d.Actor,
		PageSize: d.PageSize,
		Logger:   d.Logger,
		Inertia:  d.Inertia,
	})
	return &Feature{service: service, handler: handler}
}

func (f *Feature) Mount(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "notifications feature must be initialized")
	f.handler.Mount(router)
}

// Service exposes the feature service so other slices can emit notifications.
func (f *Feature) Service() *Service {
	assert.Assert(f != nil && f.service != nil, "notifications feature must be initialized")
	return f.service
}
