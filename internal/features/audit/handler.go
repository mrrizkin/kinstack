package audit

import (
	"errors"

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
	assert.Assert(d.Service != nil, "audit service is required")
	assert.Assert(d.Logger != nil, "audit logger is required")
	assert.Assert(d.Inertia != nil, "audit inertia adapter is required")

	return &Handler{
		service:  d.Service,
		actor:    d.Actor,
		pageSize: d.PageSize,
		logger:   d.Logger,
		inertia:  d.Inertia,
	}
}

func (h *Handler) Mount(router fiber.Router) {
	router.Get("/audit", h.index)
}

func (h *Handler) index(c fiber.Ctx) error {
	page, err := pagination.ParseRequest(c.Query("page"), c.Query("per_page"), h.defaultPerPage(c))
	if err != nil {
		return h.inertia.FailedAction(c,
			"/audit",
			fiber.StatusUnprocessableEntity,
			i18n.T("common.invalid_page"),
		)
	}
	props, err := h.service.List(c.Context(), FilterRequest{
		ActorKey: c.Query("actor_key"),
		Action:   c.Query("action"),
		Outcome:  c.Query("outcome"),
		Search:   c.Query("search"),
		From:     c.Query("from"),
		To:       c.Query("to"),
		Page:     page.Page,
		PerPage:  page.PerPage,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return h.inertia.FailedAction(c,
				"/audit",
				fiber.StatusUnprocessableEntity,
				err.Error(),
			)
		}
		h.logger.Errorw("list audit events", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return c.Render("audit/IndexView", fiber.Map{
		"entries": props.Entries,
		"filters": props.Filters,
		"page":    props.Page,
	})
}

func (h *Handler) defaultPerPage(c fiber.Ctx) int {
	session := h.actor.ActorFromContext(c)
	if session == nil {
		return pagination.DefaultPerPage
	}
	return pagination.ResolvePageSize(c.Context(), h.pageSize, session.PublicID)
}
