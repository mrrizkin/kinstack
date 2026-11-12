package system

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"

	"github.com/gofiber/fiber/v3"
)

type HandlerDependency struct {
	Service *Service
	Logger  log.Log
}

type Handler struct {
	service *Service
	logger  log.Log
}

func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "system service is required")
	assert.Assert(d.Logger != nil, "system logger is required")

	return &Handler{service: d.Service, logger: d.Logger}
}

func (h *Handler) Mount(router fiber.Router) {
	for _, surface := range []string{"403", "error", "offline", "maintenance", "help", "help/shortcuts"} {
		router.Get("/"+surface, h.page)
	}
	router.Get("/404", h.notFound)
	router.Post("/csp-report", h.cspReport)
}

const (
	// cspReportBytesMax bounds how much of a browser report is read. Anything
	// larger is discarded; the endpoint never buffers an unbounded body.
	cspReportBytesMax = 8 * 1024
	// cspLogValueMax bounds each structured field written to the log so a
	// hostile document cannot flood or forge log lines. It counts runes, not
	// bytes, so truncation never splits a UTF-8 sequence.
	cspLogValueMax = 200
)

// cspReportPayload is the subset of the CSP report contract that is safe to
// log. Unknown fields are ignored.
type cspReportPayload struct {
	Report struct {
		Directive   string `json:"effective-directive"`
		Violated    string `json:"violated-directive"`
		Disposition string `json:"disposition"`
		BlockedURI  string `json:"blocked-uri"`
		DocumentURI string `json:"document-uri"`
	} `json:"csp-report"`
}

// cspReport records a browser violation report. Browsers post these without
// authentication, so the handler reads a bounded body, extracts a few fields,
// and logs them as escaped structured values instead of the raw payload.
func (h *Handler) cspReport(c fiber.Ctx) error {
	reader := io.Reader(bytes.NewReader(c.Body()))
	if stream := c.Request().BodyStream(); stream != nil {
		reader = stream
	}
	body, err := io.ReadAll(io.LimitReader(reader, cspReportBytesMax+1))
	if err != nil {
		body = nil
	}
	truncated := len(body) > cspReportBytesMax
	if truncated {
		body = body[:cspReportBytesMax]
	}

	payload := cspReportPayload{}
	if err := json.Unmarshal(body, &payload); err != nil && !truncated {
		// A malformed report is expected noise from hostile pages, but a broken
		// reporter still has to be diagnosable.
		log.ForRequest(h.logger, c.RequestID()).
			Debugw("csp report undecodable", "error", err, "report_bytes", len(body))
	}
	directive := payload.Report.Directive
	if directive == "" {
		directive = payload.Report.Violated
	}

	log.ForRequest(h.logger, c.RequestID()).Warnw(
		"csp violation report",
		"content_type", sanitizeCSPValue(c.Get(fiber.HeaderContentType)),
		"report_bytes", len(body),
		"report_truncated", truncated,
		"directive", sanitizeCSPValue(directive),
		"disposition", sanitizeCSPValue(payload.Report.Disposition),
		"blocked_uri", sanitizeCSPValue(payload.Report.BlockedURI),
		"document_uri", sanitizeCSPValue(payload.Report.DocumentURI),
	)
	return c.SendStatus(fiber.StatusNoContent)
}

// sanitizeCSPValue removes control characters (including newlines) and caps the
// length so attacker-controlled text cannot forge or bloat log entries.
func sanitizeCSPValue(value string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return ' '
		}
		return r
	}, value)
	cleaned = strings.Join(strings.Fields(cleaned), " ")
	runes := []rune(cleaned)
	if len(runes) > cspLogValueMax {
		cleaned = string(runes[:cspLogValueMax])
	}
	return cleaned
}

func (h *Handler) notFound(c fiber.Ctx) error {
	props := h.service.NotFound()
	err := c.Status(fiber.StatusNotFound).Render("system/NotFoundView", fiber.Map{
		"status":  props.Status,
		"message": props.Message,
	})
	if err != nil {
		log.ForRequest(h.logger, c.RequestID()).Errorw("render system not-found page", "error", err)
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
		log.ForRequest(h.logger, c.RequestID()).Errorw("render system state page", "error", err)
	}
	return err
}
