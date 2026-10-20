package users

import (
	"errors"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/authz"
	platforminertia "kinstack/internal/platform/inertia"

	"codeberg.org/mrrizkin/fibertia"
	"github.com/gofiber/fiber/v3"
)

// HandlerDependency contains the service, actor boundary, Inertia adapter, and
// logger used by the handler.
type HandlerDependency struct {
	Service *Service
	Actor   *actor.Actor
	Logger  log.Log
	Inertia *fibertia.Fibertia
}

// Handler owns the HTTP boundary for user management.
type Handler struct {
	service *Service
	actor   *actor.Actor
	logger  log.Log
	inertia *fibertia.Fibertia
}

// NewHandler constructs the users HTTP handler.
func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "users service is required")
	assert.Assert(d.Actor != nil, "users actor boundary is required")
	assert.Assert(d.Logger != nil, "users logger is required")
	assert.Assert(d.Inertia != nil, "users inertia adapter is required")

	return &Handler{
		service: d.Service,
		actor:   d.Actor,
		logger:  d.Logger,
		inertia: d.Inertia,
	}
}

func (h *Handler) usersPath() string {
	return "/users"
}

func (h *Handler) accessPath() string {
	return "/access"
}

// Mount registers user management and access control routes on the guarded group.
func (h *Handler) Mount(router fiber.Router) {
	router.Get("/users", h.index)
	router.Post("/users", h.create)
	router.Post("/users/invite", h.invite)
	router.Put("/users/:publicID", h.update)
	router.Post("/users/:publicID/password", h.resetPassword)
	router.Post("/users/:publicID/active", h.setActive)

	router.Get("/access", h.access)
	router.Post("/access/roles", h.createRole)
	router.Put("/access/roles/:code", h.updateRole)
	router.Post("/access/roles/:code/delete", h.deleteRole)
	router.Post("/access/roles/:code/permissions", h.setRolePermissions)
	router.Post("/access/users/:publicID/role", h.assignRole)
	router.Post("/access/users/:publicID/permissions", h.setOverride)
	router.Post("/access/users/:publicID/permissions/delete", h.deleteOverride)
}

func (h *Handler) index(c fiber.Ctx) error {
	items, err := h.service.List(c.Context())
	if err != nil {
		h.logger.Errorw("list users", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	roles, err := h.service.RoleOptions(c.Context())
	if err != nil {
		h.logger.Errorw("list roles", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return c.Render("users/IndexView", fiber.Map{"users": items, "roles": roles})
}

func (h *Handler) access(c fiber.Ctx) error {
	props, err := h.service.AccessView(c.Context())
	if err != nil {
		h.logger.Errorw("access view", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return c.Render("access/IndexView", fiber.Map{
		"roles":       props.Roles,
		"permissions": props.Permissions,
		"categories":  props.Categories,
		"users":       props.Users,
	})
}

func (h *Handler) writeError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrUsernameTaken):
		return h.validationFields(c, map[string]string{"username": err.Error()})
	case errors.Is(err, ErrInvalidRole), errors.Is(err, authz.ErrInvalidRole):
		return h.validationFields(c, map[string]string{"role": err.Error()})
	case errors.Is(err, ErrLastAdministrator):
		return h.validationFields(c, map[string]string{"is_active": err.Error()})
	case errors.Is(err, ErrInvalidInput),
		errors.Is(err, ErrAuthorizationUnavailable),
		errors.Is(err, authz.ErrRoleCodeTaken),
		errors.Is(err, authz.ErrSystemRole),
		errors.Is(err, authz.ErrRoleInUse),
		errors.Is(err, authz.ErrUnknownPermission),
		errors.Is(err, authz.ErrInvalidOverride),
		errors.Is(err, authz.ErrUserNotFound):
		return h.validationError(c, err.Error())
	case errors.Is(err, ErrNotFound), errors.Is(err, authz.ErrRoleNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	default:
		h.logger.Errorw("user mutation failed", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
}

func (h *Handler) validationError(c fiber.Ctx, message string) error {
	return h.validationFields(c, map[string]string{"form": message})
}

func (h *Handler) validationFields(c fiber.Ctx, fields map[string]string) error {
	back := c.Get("Referer")
	if back == "" {
		back = h.usersPath()
	}
	return platforminertia.ValidationErrors(h.inertia, c, back, fields)
}
