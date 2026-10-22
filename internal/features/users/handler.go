package users

import (
	"context"
	"errors"
	"net/url"
	"strconv"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"
	"kinstack/internal/pkg/pagination"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/authz"
	"kinstack/internal/platform/i18n"
	platforminertia "kinstack/internal/platform/inertia"

	"github.com/gofiber/fiber/v3"
)

// HandlerDependency contains the boundaries used by the handler.
type HandlerDependency struct {
	Service  *Service
	Actor    *actor.Actor
	PageSize pagination.PageSizeReader
	Logger   log.Log
	Inertia  *platforminertia.Adapter
}

// Handler owns the HTTP boundary for user management.
type Handler struct {
	service  *Service
	actor    *actor.Actor
	pageSize pagination.PageSizeReader
	logger   log.Log
	inertia  *platforminertia.Adapter
}

// NewHandler constructs the users HTTP handler.
func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "users service is required")
	assert.Assert(d.Actor != nil, "users actor boundary is required")
	assert.Assert(d.Logger != nil, "users logger is required")
	assert.Assert(d.Inertia != nil, "users inertia adapter is required")

	return &Handler{
		service:  d.Service,
		actor:    d.Actor,
		pageSize: d.PageSize,
		logger:   d.Logger,
		inertia:  d.Inertia,
	}
}

// context attaches the acting principal so authorization mutations can enforce
// delegation rules (rank and permission subset) inside the platform service.
func (h *Handler) context(c fiber.Ctx) context.Context {
	if session := h.actor.ActorFromContext(c); session != nil {
		return authz.WithPrincipal(c.Context(), authz.PrincipalFromActor(session))
	}
	return c.Context()
}

func (h *Handler) defaultPerPage(c fiber.Ctx) int {
	session := h.actor.ActorFromContext(c)
	if session == nil {
		return pagination.DefaultPerPage
	}
	return pagination.ResolvePageSize(c.Context(), h.pageSize, session.PublicID)
}

func (h *Handler) usersPath() string {
	return "/users"
}

// accessPathFor keeps the user on the access page they were viewing after a
// successful mutation. The two tabs page independently, so both hints are
// preserved; invalid values fall back to the first page.
func (h *Handler) accessPathFor(c fiber.Ctx) string {
	params := url.Values{}
	if page := pagination.PageHint(c.Query("page")); page > 1 {
		params.Set("page", strconv.Itoa(page))
	}
	if page := pagination.PageHint(c.Query("roles_page")); page > 1 {
		params.Set("roles_page", strconv.Itoa(page))
	}
	if len(params) == 0 {
		return h.accessPath()
	}
	return h.accessPath() + "?" + params.Encode()
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
	query, err := ParsePageQuery(c.Query("page"), c.Query("per_page"), h.defaultPerPage(c))
	if err != nil {
		return h.invalidPagination(c, h.usersPath())
	}
	list, err := h.service.List(c.Context(), query)
	if err != nil {
		h.logger.Errorw("list users", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	roles, err := h.service.RoleOptions(c.Context())
	if err != nil {
		h.logger.Errorw("list roles", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return c.Render("users/IndexView", fiber.Map{
		"users": list.Users,
		"roles": roles,
		"page":  list.Page,
	})
}

func (h *Handler) access(c fiber.Ctx) error {
	query, err := ParsePageQuery(c.Query("page"), c.Query("per_page"), h.defaultPerPage(c))
	if err != nil {
		return h.invalidPagination(c, h.accessPath())
	}
	rolesQuery, err := ParsePageQuery(
		c.Query("roles_page"),
		c.Query("per_page"),
		h.defaultPerPage(c),
	)
	if err != nil {
		return h.invalidPagination(c, h.accessPath())
	}
	props, err := h.service.AccessView(c.Context(), query, rolesQuery)
	if err != nil {
		h.logger.Errorw("access view", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return c.Render("access/IndexView", fiber.Map{
		"roles":       props.Roles,
		"roles_page":  props.RolesPage,
		"permissions": props.Permissions,
		"categories":  props.Categories,
		"users":       props.Users,
		"page":        props.Page,
	})
}

func (h *Handler) writeError(c fiber.Ctx, err error) error {
	return h.writeErrorAt(c, h.usersPath(), err)
}

// writeErrorAt reports a mutation failure and redirects validation errors back
// to the page that produced them. The caller supplies the path; the Referer
// header is never trusted.
func (h *Handler) writeErrorAt(c fiber.Ctx, back string, err error) error {
	switch {
	case errors.Is(err, ErrUsernameTaken):
		return h.validationFieldsAt(c, back, map[string]string{"username": err.Error()})
	case errors.Is(err, ErrInvalidRole), errors.Is(err, authz.ErrInvalidRole):
		return h.validationFieldsAt(c, back, map[string]string{"role": err.Error()})
	case errors.Is(err, ErrLastAdministrator):
		return h.validationFieldsAt(c, back, map[string]string{"is_active": err.Error()})
	case errors.Is(err, ErrForbidden),
		errors.Is(err, ErrSelfChange),
		errors.Is(err, authz.ErrPrivilegeEscalation),
		errors.Is(err, authz.ErrSelfChange),
		errors.Is(err, authz.ErrManagementSession):
		return h.inertia.FailedAction(c,
			back,
			fiber.StatusForbidden,
			err.Error(),
		)
	case errors.Is(err, ErrInvalidInput),
		errors.Is(err, ErrAuthorizationUnavailable),
		errors.Is(err, authz.ErrRoleCodeTaken),
		errors.Is(err, authz.ErrSystemRole),
		errors.Is(err, authz.ErrRoleInUse),
		errors.Is(err, authz.ErrUnknownPermission),
		errors.Is(err, authz.ErrInvalidOverride),
		errors.Is(err, authz.ErrUserNotFound):
		return h.validationFieldsAt(c, back, map[string]string{"form": err.Error()})
	case errors.Is(err, ErrNotFound), errors.Is(err, authz.ErrRoleNotFound):
		return h.inertia.FailedAction(c,
			back,
			fiber.StatusNotFound,
			err.Error(),
		)
	default:
		h.logger.Errorw("user mutation failed", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
}

func (h *Handler) validationFieldsAt(c fiber.Ctx, back string, fields map[string]string) error {
	return h.inertia.ValidationErrors(c, back, fields)
}

// invalidPagination answers a malformed page request. JSON clients get the
// usual 422 envelope; browser navigation returns with a flash message.
func (h *Handler) invalidPagination(c fiber.Ctx, back string) error {
	message := i18n.T("common.invalid_page")
	if platforminertia.WantsJSON(c) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": message})
	}
	return h.inertia.RedirectWithFlash(c, back, message)
}
