package audit

import (
	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/pkg/pagination"
	"kinstack/internal/platform/actor"
	platforminertia "kinstack/internal/platform/inertia"

	"github.com/gofiber/fiber/v3"
)

// Dependency contains the persistence boundary and logger used by the feature.
type Dependency struct {
	Repository Repository
	Actor      *actor.Actor
	PageSize   pagination.PageSizeReader
	Logger     applog.Log
	Inertia    *platforminertia.Adapter
}

// Feature owns the audit viewer service and HTTP entry point.
type Feature struct {
	service *Service
	handler *Handler
}

// New constructs the audit feature from explicit dependencies.
func New(d Dependency) *Feature {
	assert.Assert(d.Repository != nil, "audit repository is required")
	assert.Assert(d.Logger != nil, "audit logger is required")
	assert.Assert(d.Inertia != nil, "audit inertia adapter is required")

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

// Mount registers the audit viewer routes on the authenticated workspace.
func (f *Feature) Mount(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "audit feature must be initialized")
	f.handler.Mount(router)
}

// Service exposes the feature service for composition-root adapters.
func (f *Feature) Service() *Service {
	assert.Assert(f != nil && f.service != nil, "audit feature must be initialized")
	return f.service
}
