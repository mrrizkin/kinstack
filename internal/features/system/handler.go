package system

import (
	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"

	"github.com/gofiber/fiber/v3"
)

// HandlerDependency contains the service and logging boundaries used by the handler.
type HandlerDependency struct {
	Service *Service
	Logger  log.Log
}

// Handler owns the HTTP boundary for system pages.
type Handler struct {
	service *Service
	logger  log.Log
}

// NewHandler constructs the system HTTP handler.
func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "system service is required")
	assert.Assert(d.Logger != nil, "system logger is required")

	return &Handler{service: d.Service, logger: d.Logger}
}

// Mount registers the existing public system routes.
func (h *Handler) Mount(router fiber.Router) {
	for _, surface := range []string{"403", "error", "offline", "maintenance", "help", "help/shortcuts"} {
		router.Get("/"+surface, h.page)
	}
	router.Get("/404", h.notFound)
	router.Post("/csp-report", h.cspReport)
}

const maxCSPReportBytes = 8 * 1024

func (h *Handler) cspReport(c fiber.Ctx) error {
	body := c.Body()
	if len(body) > maxCSPReportBytes {
		body = body[:maxCSPReportBytes]
	}
	h.logger.Warnw(
		"csp violation report",
		"request_id", c.RequestID(),
		"content_type", c.Get(fiber.HeaderContentType),
		"report", string(body),
	)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) notFound(c fiber.Ctx) error {
	props := h.service.NotFound()
	err := c.Status(fiber.StatusNotFound).Render("system/NotFoundView", fiber.Map{
		"status":  props.Status,
		"message": props.Message,
	})
	if err != nil {
		h.logger.Errorw("render system not-found page", "request_id", c.RequestID(), "error", err)
	}
	return err
}

func (h *Handler) page(c fiber.Ctx) error {
	rawSurface := c.Path()[1:]
	surface := "system-" + rawSurface
	if rawSurface == "403" {
		surface = "system-forbidden"
	} else if rawSurface == "help/shortcuts" {
		surface = "system-shortcuts"
	}
	props := h.service.Page(surface)
	component := "system/SystemStateView"
	if rawSurface == "help" || rawSurface == "help/shortcuts" {
		component = "system/HelpView"
	}
	err := c.Render(component, fiber.Map{
		"surface":   props.Surface,
		"topics":    props.Topics,
		"shortcuts": props.Shortcuts,
	})
	if err != nil {
		h.logger.Errorw("render system state page", "request_id", c.RequestID(), "error", err)
	}
	return err
}
