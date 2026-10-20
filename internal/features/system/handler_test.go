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
	fields  map[string]any
}

func (l *captureLogger) Warnw(message string, keysAndValues ...any) {
	l.message = message
	l.fields = map[string]any{}
	for i := 0; i+1 < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		l.fields[key] = keysAndValues[i+1]
	}
}

func (l *captureLogger) field(key string) string {
	value, _ := l.fields[key].(string)
	return value
}

func newCSPTestApp(t *testing.T, logger *captureLogger) *fiber.App {
	t.Helper()
	handler := NewHandler(HandlerDependency{
		Service: NewService(ServiceDependency{Repository: NewStaticRepository()}),
		Logger:  logger,
	})
	app := fiber.New()
	handler.Mount(app)
	return app
}

func postCSPReport(t *testing.T, app *fiber.App, body string) int {
	t.Helper()
	req := httptest.NewRequest("POST", "/csp-report", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/csp-report")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	return resp.StatusCode
}

func TestCSPReportLogsStructuredFieldsOnly(t *testing.T) {
	logger := &captureLogger{}
	app := newCSPTestApp(t, logger)

	body := `{"csp-report":{"effective-directive":"script-src-elem",` +
		`"blocked-uri":"https://evil.example/a.js\nFAKE LOG LINE\r",` +
		`"document-uri":"https://app.example/dashboard","disposition":"enforce"}}`
	if status := postCSPReport(t, app, body); status != fiber.StatusNoContent {
		t.Fatalf("status=%d, want %d", status, fiber.StatusNoContent)
	}
	if logger.message != "csp violation report" {
		t.Fatalf("unexpected log message: %q", logger.message)
	}
	if got := logger.field("directive"); got != "script-src-elem" {
		t.Fatalf("directive=%q", got)
	}
	if got := logger.field("disposition"); got != "enforce" {
		t.Fatalf("disposition=%q", got)
	}
	if got := logger.field("document_uri"); got != "https://app.example/dashboard" {
		t.Fatalf("document_uri=%q", got)
	}
	blocked := logger.field("blocked_uri")
	if blocked == "" {
		t.Fatal("blocked_uri must be logged")
	}
	if strings.ContainsAny(blocked, "\r\n\x00") {
		t.Fatalf("blocked_uri must not contain control characters: %q", blocked)
	}
	if len(blocked) > maxCSPLogValue {
		t.Fatalf("blocked_uri must be bounded, got %d bytes", len(blocked))
	}
	if _, ok := logger.fields["report"]; ok {
		t.Fatal("raw report payload must not be logged")
	}
}

func TestCSPReportTruncatesOversizedBodies(t *testing.T) {
	logger := &captureLogger{}
	app := newCSPTestApp(t, logger)

	body := strings.Repeat("x", maxCSPReportBytes+512)
	if status := postCSPReport(t, app, body); status != fiber.StatusNoContent {
		t.Fatalf("status=%d, want %d", status, fiber.StatusNoContent)
	}
	if got, _ := logger.fields["report_truncated"].(bool); !got {
		t.Fatal("report_truncated must be true")
	}
	if got, _ := logger.fields["report_bytes"].(int); got != maxCSPReportBytes {
		t.Fatalf("report_bytes=%d, want %d", got, maxCSPReportBytes)
	}
	if got := logger.field("directive"); got != "" {
		t.Fatalf("truncated payload must not produce a directive, got %q", got)
	}
}

func TestSanitizeCSPValueBoundsLength(t *testing.T) {
	got := sanitizeCSPValue(strings.Repeat("a", maxCSPLogValue+50))
	if len(got) != maxCSPLogValue {
		t.Fatalf("len=%d, want %d", len(got), maxCSPLogValue)
	}
	if got := sanitizeCSPValue("a\tb\nc\x00d"); got != "a b c d" {
		t.Fatalf("unexpected sanitized value: %q", got)
	}
}
