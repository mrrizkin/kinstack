package notifications

import (
	"errors"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/i18n"
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

// Handler owns the HTTP boundary for the notification center.
type Handler struct {
	service *Service
	actor   *actor.Actor
	logger  log.Log
	inertia *fibertia.Fibertia
}

// NewHandler constructs the notification HTTP handler.
func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "notification service is required")
	assert.Assert(d.Actor != nil, "notification actor boundary is required")
	assert.Assert(d.Logger != nil, "notification logger is required")
	assert.Assert(d.Inertia != nil, "notification inertia adapter is required")

	return &Handler{service: d.Service, actor: d.Actor, logger: d.Logger, inertia: d.Inertia}
}

func (h *Handler) notificationsPath() string {
	return "/notifications"
}

// Mount registers notification routes on the guarded group.
func (h *Handler) Mount(router fiber.Router) {
	router.Get("/notifications", h.index)
	router.Get("/notifications/feed", h.feed)
	router.Post("/notifications/read-all", h.markAllRead)
	router.Post("/notifications/:publicID/read", h.markRead)
}

func (h *Handler) index(c fiber.Ctx) error {
	props, err := h.service.List(c.Context(), h.recipient(c))
	if err != nil {
		return h.internalServerError(c, "list notifications", err)
	}
	return c.Render("notifications/IndexView", fiber.Map{
		"notifications": props.Notifications,
		"unread":        props.Unread,
	})
}

// feed returns the compact notification payload used by the dashboard
// slide-over, so the shell can refresh without a full Inertia visit.
func (h *Handler) feed(c fiber.Ctx) error {
	props, err := h.service.List(c.Context(), h.recipient(c))
	if err != nil {
		return h.internalServerError(c, "list notification feed", err)
	}
	return c.JSON(props)
}

func (h *Handler) markRead(c fiber.Ctx) error {
	if err := h.service.MarkRead(c.Context(), h.recipient(c), c.Params("publicID")); err != nil {
		if errors.Is(err, ErrNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, ErrInvalidInput) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		return h.internalServerError(c, "mark notification read", err)
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.notificationsPath(), i18n.T("notifications.read"))
}

func (h *Handler) markAllRead(c fiber.Ctx) error {
	if err := h.service.MarkAllRead(c.Context(), h.recipient(c)); err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		return h.internalServerError(c, "mark all notifications read", err)
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.notificationsPath(), i18n.T("notifications.all_read"))
}

func (h *Handler) recipient(c fiber.Ctx) string {
	sessionActor := h.actor.ActorFromContext(c)
	if sessionActor == nil {
		return ""
	}
	return sessionActor.PublicID
}

func (h *Handler) internalServerError(c fiber.Ctx, message string, err error) error {
	h.logger.Errorw(message, "request_id", c.RequestID(), "error", err)
	return fiber.ErrInternalServerError
}
