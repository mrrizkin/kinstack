package audit

import (
	"errors"
	"net/http"
	"strings"

	platformactor "kinstack/internal/platform/actor"

	"github.com/gofiber/fiber/v3"
)

// -----------------------------------------------------------------------------
// HTTP middleware
// -----------------------------------------------------------------------------

// Middleware records mutations and selected protected reads. Storage failures
// are reported through onError without changing the request result.
func Middleware(
	recorder Recorder,
	actorBoundary *platformactor.Actor,
	onError func(error),
) fiber.Handler {
	if recorder == nil {
		return func(c fiber.Ctx) error { return c.Next() }
	}
	return func(c fiber.Ctx) error {
		if !shouldAudit(c) {
			return c.Next()
		}
		err := c.Next()
		status := responseStatus(c, err)
		event := Event{
			RequestID: c.RequestID(),
			Action:    auditAction(c.Method()),
			Resource:  resourcePath(c),
			Method:    c.Method(),
			Status:    status,
			Outcome:   outcome(status),
			Metadata:  map[string]string{"source": "http"},
		}
		if actor := actorBoundary.ActorFromContext(c); actor != nil {
			event.ActorKey = actor.PublicID
		}
		if recordErr := recorder.Record(c.Context(), event); recordErr != nil && onError != nil {
			onError(recordErr)
		}
		return err
	}
}

// -----------------------------------------------------------------------------
// Event helpers
// -----------------------------------------------------------------------------

func shouldAudit(c fiber.Ctx) bool {
	path := c.Path()
	if strings.HasPrefix(path, "/build/") || strings.HasPrefix(path, "/uploads/") ||
		path == "/health" || path == "/csp-report" {
		return false
	}
	if c.Method() != http.MethodGet && c.Method() != http.MethodHead &&
		c.Method() != http.MethodOptions {
		return true
	}
	return strings.HasPrefix(path, "/profile") ||
		strings.HasPrefix(path, "/users") ||
		strings.HasPrefix(path, "/access")
}

func auditAction(method string) Action {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return ActionHTTPRead
	default:
		return ActionHTTPMutate
	}
}

func resourcePath(c fiber.Ctx) string {
	if route := c.Route(); route != nil && route.Path != "" {
		return route.Path
	}
	return c.Path()
}

func outcome(status int) Outcome {
	if status >= http.StatusBadRequest {
		return OutcomeFailure
	}
	return OutcomeSuccess
}

func responseStatus(c fiber.Ctx, err error) int {
	if err != nil {
		var statusCoder interface{ StatusCode() int }
		if errors.As(err, &statusCoder) && statusCoder.StatusCode() > 0 {
			return statusCoder.StatusCode()
		}
		var fiberErr *fiber.Error
		if errors.As(err, &fiberErr) && fiberErr.Code > 0 {
			return fiberErr.Code
		}
	}
	return c.Response().StatusCode()
}
