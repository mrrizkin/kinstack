package log

import (
	"bufio"
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/rs/zerolog"
)

// TestHTTPMiddlewareLogsRequestID is the gate behind the observability rule:
// every server log line names the request it belongs to. Without the request
// ID, two interleaved failures are indistinguishable in the log.
func TestHTTPMiddlewareLogsRequestID(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := &Logger{core: zerolog.New(buffer).With().Timestamp().Logger()}

	app := fiber.New()
	app.Use(requestid.New())
	app.Use(HTTPMiddleware(logger))
	app.Get("/ok", func(c fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/boom", func(c fiber.Ctx) error { return fiber.ErrTeapot })

	for _, path := range []string{"/ok", "/boom"} {
		response, err := app.Test(httptest.NewRequest(fiber.MethodGet, path, nil))
		if err != nil {
			t.Fatalf("%s: request failed: %v", path, err)
		}
		_ = response.Body.Close()
	}

	lines := 0
	scanner := bufio.NewScanner(buffer)
	for scanner.Scan() {
		raw := scanner.Bytes()
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		lines++
		fields := map[string]any{}
		if err := json.Unmarshal(raw, &fields); err != nil {
			t.Fatalf("log line is not JSON: %q", raw)
		}
		requestID, ok := fields["request_id"].(string)
		if !ok || requestID == "" {
			t.Fatalf("log line carries no request_id: %s", raw)
		}
	}
	if lines < 4 {
		t.Fatalf("expected a start and completion line per request, got %d lines", lines)
	}
}
