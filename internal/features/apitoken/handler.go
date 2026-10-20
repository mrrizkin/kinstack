package apitoken

import (
	"errors"

	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/authz"
	"kinstack/internal/platform/i18n"
	platforminertia "kinstack/internal/platform/inertia"
	platformsession "kinstack/internal/platform/session"

	"codeberg.org/mrrizkin/fibertia"
	"github.com/gofiber/fiber/v3"
)

// createdTokenFlashKey carries the one-time plaintext token across the create
// redirect so it can be shown exactly once.
const createdTokenFlashKey = "created_api_token"

// HandlerDependency contains the service, actor boundary, Inertia adapter, and
// logger used by the handler.
type HandlerDependency struct {
	Service       *Service
	Actor         *actor.Actor
	Authorization *authz.Service
	Logger        applog.Log
	Inertia       *fibertia.Fibertia
}

// Handler owns the HTTP boundary for personal API tokens.
type Handler struct {
	service       *Service
	actor         *actor.Actor
	authorization *authz.Service
	logger        applog.Log
	inertia       *fibertia.Fibertia
}

// NewHandler constructs the API token HTTP handler.
func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "api token service is required")
	assert.Assert(d.Actor != nil, "api token actor boundary is required")
	assert.Assert(d.Authorization != nil, "api token authorization service is required")
	assert.Assert(d.Logger != nil, "api token logger is required")
	assert.Assert(d.Inertia != nil, "api token inertia adapter is required")

	return &Handler{
		service:       d.Service,
		actor:         d.Actor,
		authorization: d.Authorization,
		logger:        d.Logger,
		inertia:       d.Inertia,
	}
}

// Mount registers the API token routes on the authenticated workspace.
func (h *Handler) Mount(router fiber.Router) {
	router.Get("/api-tokens", h.show)
	router.Post("/api-tokens", h.create)
	router.Post("/api-tokens/:publicID/revoke", h.revoke)
}

func (h *Handler) actorPublicID(c fiber.Ctx) string {
	if session := h.actor.ActorFromContext(c); session != nil {
		return session.PublicID
	}
	return ""
}

// isSuperuser reports the ephemeral break-glass actor. It has no persisted
// profile, so personal API tokens cannot authenticate as that actor.
func (h *Handler) isSuperuser(c fiber.Ctx) bool {
	session := h.actor.ActorFromContext(c)
	return session != nil && session.Superuser
}

func (h *Handler) show(c fiber.Ctx) error {
	if h.isSuperuser(c) {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard")
	}
	tokens, err := h.service.List(c.Context(), h.actorPublicID(c))
	if err != nil {
		h.logger.Errorw("list api tokens", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	props := fiber.Map{"tokens": tokens}
	if created, ok := platformsession.ConsumeFlash(c, createdTokenFlashKey); ok {
		props["created_token"] = created
	}
	return c.Render("apitoken/IndexView", props)
}

func (h *Handler) create(c fiber.Ctx) error {
	if h.isSuperuser(c) {
		return fiber.ErrForbidden
	}
	var request CreateAPITokenRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": i18n.T("common.invalid_payload")})
	}
	if err := h.validateScopes(c, request.Scopes); err != nil {
		return err
	}
	created, err := h.service.Create(
		c.Context(),
		h.actorPublicID(c),
		request.Name,
		request.ExpiresInDays,
		request.Scopes,
	)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return platforminertia.ValidationErrors(
				h.inertia,
				c,
				"/api-tokens",
				map[string]string{"name": err.Error()},
			)
		}
		h.logger.Errorw("create api token", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	if err := platformsession.SetFlash(c, createdTokenFlashKey, created.Token); err != nil {
		h.logger.Errorw("store created api token", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To("/api-tokens")
}

// validateScopes rejects scope codes that exceed the caller's own permissions,
// so a token can never grant more than its owner currently holds.
func (h *Handler) validateScopes(c fiber.Ctx, scopes []string) error {
	if len(scopes) == 0 {
		return nil
	}
	session := h.actor.ActorFromContext(c)
	if session == nil {
		return fiber.ErrUnauthorized
	}
	granted, err := h.authorization.Permissions(c.Context(), authz.PrincipalFromActor(session))
	if err != nil {
		h.logger.Errorw("resolve permissions for token scopes", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	allowed := make(map[string]struct{}, len(granted))
	for _, permission := range granted {
		allowed[string(permission)] = struct{}{}
	}
	for _, scope := range scopes {
		if _, ok := allowed[scope]; !ok {
			return platforminertia.ValidationErrors(h.inertia, c, "/api-tokens", map[string]string{
				"scopes": i18n.T("auth.scope_exceeded", scope),
			})
		}
	}
	return nil
}

func (h *Handler) revoke(c fiber.Ctx) error {
	if h.isSuperuser(c) {
		return fiber.ErrForbidden
	}
	if err := h.service.Revoke(c.Context(), h.actorPublicID(c), c.Params("publicID")); err != nil {
		if !errors.Is(err, ErrNotFound) {
			h.logger.Errorw("revoke api token", "request_id", c.RequestID(), "error", err)
			return fiber.ErrInternalServerError
		}
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, "/api-tokens", i18n.T("apitoken.revoked"))
}
