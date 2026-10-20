package system

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	applog "kinstack/internal/pkg/log"

	"github.com/gofiber/fiber/v3"
)

type captureLogger struct {
	applog.Log
	message string
	report  string
}

func (l *captureLogger) Warnw(message string, keysAndValues ...any) {
	l.message = message
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		if keysAndValues[i] == "report" {
			l.report, _ = keysAndValues[i+1].(string)
		}
	}
}

func TestCSPReportIsAcceptedAndBounded(t *testing.T) {
	logger := &captureLogger{}
	handler := NewHandler(HandlerDependency{
		Service: NewService(ServiceDependency{Repository: NewStaticRepository()}),
		Logger:  logger,
	})
	app := fiber.New()
	handler.Mount(app)

	body := strings.Repeat("x", maxCSPReportBytes+100)
	req := httptest.NewRequest("POST", "/csp-report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/csp-report")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != fiber.StatusNoContent {
		t.Fatalf("status=%d, want %d", resp.StatusCode, fiber.StatusNoContent)
	}
	if logger.message != "csp violation report" {
		t.Fatalf("unexpected log message: %q", logger.message)
	}
	if len(logger.report) != maxCSPReportBytes {
		t.Fatalf("report length=%d, want %d", len(logger.report), maxCSPReportBytes)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
}
