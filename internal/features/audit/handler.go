package audit

import (
	"errors"
	"strconv"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"
	platforminertia "kinstack/internal/platform/inertia"

	"github.com/gofiber/fiber/v3"
)

// HandlerDependency contains the service and logging boundaries used by the handler.
type HandlerDependency struct {
	Service *Service
	Logger  log.Log
	Inertia *platforminertia.Adapter
}

// Handler owns the HTTP boundary for the audit viewer.
type Handler struct {
	service *Service
	logger  log.Log
	inertia *platforminertia.Adapter
}

// NewHandler constructs the audit HTTP handler.
func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "audit service is required")
	assert.Assert(d.Logger != nil, "audit logger is required")
	assert.Assert(d.Inertia != nil, "audit inertia adapter is required")

	return &Handler{service: d.Service, logger: d.Logger, inertia: d.Inertia}
}

// Mount registers the audit viewer routes on the guarded group.
func (h *Handler) Mount(router fiber.Router) {
	router.Get("/audit", h.index)
}

func (h *Handler) index(c fiber.Ctx) error {
	props, err := h.service.List(c.Context(), FilterRequest{
		ActorKey: c.Query("actor_key"),
		Action:   c.Query("action"),
		Outcome:  c.Query("outcome"),
		Search:   c.Query("search"),
		From:     c.Query("from"),
		To:       c.Query("to"),
		Page:     queryInt(c, "page"),
		PerPage:  queryInt(c, "per_page"),
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
		"entries":  props.Entries,
		"filters":  props.Filters,
		"total":    props.Total,
		"page":     props.Page,
		"per_page": props.PerPage,
		"pages":    props.Pages,
	})
}

func queryInt(c fiber.Ctx, key string) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return 0
	}
	return value
}
