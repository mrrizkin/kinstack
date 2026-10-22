package notifications

import (
	"errors"
	"strconv"
	"strings"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"
	"kinstack/internal/pkg/pagination"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/i18n"
	platforminertia "kinstack/internal/platform/inertia"

	"github.com/gofiber/fiber/v3"
)

type HandlerDependency struct {
	Service  *Service
	Actor    *actor.Actor
	PageSize pagination.PageSizeReader
	Logger   log.Log
	Inertia  *platforminertia.Adapter
}

type Handler struct {
	service  *Service
	actor    *actor.Actor
	pageSize pagination.PageSizeReader
	logger   log.Log
	inertia  *platforminertia.Adapter
}

func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "notification service is required")
	assert.Assert(d.Actor != nil, "notification actor boundary is required")
	assert.Assert(d.Logger != nil, "notification logger is required")
	assert.Assert(d.Inertia != nil, "notification inertia adapter is required")

	return &Handler{
		service:  d.Service,
		actor:    d.Actor,
		pageSize: d.PageSize,
		logger:   d.Logger,
		inertia:  d.Inertia,
	}
}

func (h *Handler) notificationsPath() string {
	return "/notifications"
}

func (h *Handler) Mount(router fiber.Router) {
	router.Get("/notifications", h.index)
	router.Get("/notifications/feed", h.feed)
	router.Post("/notifications/read-all", h.markAllRead)
	router.Post("/notifications/:publicID/read", h.markRead)
}

func (h *Handler) index(c fiber.Ctx) error {
	props, err := h.service.List(c.Context(), h.recipient(c), "", h.pageLimit(c))
	if err != nil {
		return h.internalServerError(c, "list notifications", err)
	}
	return c.Render("notifications/IndexView", fiber.Map{
		"notifications": props.Notifications,
		"unread":        props.Unread,
		"next_cursor":   props.NextCursor,
	})
}

// feed returns the compact notification payload used by the dashboard
// slide-over, so the shell can refresh without a full Inertia visit.
func (h *Handler) feed(c fiber.Ctx) error {
	limit := h.pageLimit(c)
	if raw := strings.TrimSpace(c.Query("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			return c.Status(fiber.StatusUnprocessableEntity).
				JSON(fiber.Map{"error": i18n.T("common.invalid_page")})
		}
		limit = parsed
	}
	props, err := h.service.List(c.Context(), h.recipient(c), c.Query("cursor"), limit)
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return c.Status(fiber.StatusUnprocessableEntity).
				JSON(fiber.Map{"error": i18n.T("common.invalid_page")})
		}
		return h.internalServerError(c, "list notification feed", err)
	}
	return c.JSON(props)
}

func (h *Handler) pageLimit(c fiber.Ctx) int {
	if sessionActor := h.actor.ActorFromContext(c); sessionActor != nil {
		return pagination.ResolvePageSize(c.Context(), h.pageSize, sessionActor.PublicID)
	}
	return pagination.DefaultPerPage
}

func (h *Handler) markRead(c fiber.Ctx) error {
	if err := h.service.MarkRead(c.Context(), h.recipient(c), c.Params("publicID")); err != nil {
		switch {
		case errors.Is(err, ErrNotFound):
			return h.inertia.FailedAction(
				c,
				h.notificationsPath(),
				fiber.StatusNotFound,
				err.Error(),
			)
		case errors.Is(err, ErrInvalidInput):
			return h.inertia.FailedAction(
				c,
				h.notificationsPath(),
				fiber.StatusUnprocessableEntity,
				err.Error(),
			)
		}
		return h.internalServerError(c, "mark notification read", err)
	}
	if platforminertia.WantsJSON(c) {
		return c.SendStatus(fiber.StatusNoContent)
	}
	return h.inertia.RedirectWithFlash(c,
		h.notificationsPath(),
		i18n.T("notifications.read"),
	)
}

func (h *Handler) markAllRead(c fiber.Ctx) error {
	if err := h.service.MarkAllRead(c.Context(), h.recipient(c)); err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return h.inertia.FailedAction(
				c,
				h.notificationsPath(),
				fiber.StatusUnprocessableEntity,
				err.Error(),
			)
		}
		return h.internalServerError(c, "mark all notifications read", err)
	}
	if platforminertia.WantsJSON(c) {
		return c.SendStatus(fiber.StatusNoContent)
	}
	return h.inertia.RedirectWithFlash(c,
		h.notificationsPath(),
		i18n.T("notifications.all_read"),
	)
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
