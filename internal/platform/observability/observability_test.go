package observability

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"kinstack/config"

	"github.com/gofiber/fiber/v3"
)

func TestTelemetryExportsToOpenObserve(t *testing.T) {
	var mu sync.Mutex
	paths := map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths[r.URL.Path] = r.Header.Get("Authorization")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	cfg := config.Config{
		OpenObserve: config.OpenObserveConfig{
			Enabled:          true,
			Endpoint:         server.URL,
			Organization:     "kinstack",
			AuthHeader:       "Basic dGVzdDp0ZXN0",
			StreamName:       "kinstack",
			ServiceName:      "kinstack",
			Environment:      "test",
			TracesSampleRate: 1,
			ExportLogs:       true,
			ExportMetrics:    true,
		},
	}

	telemetry, err := New(Dependency{Config: cfg})
	if err != nil {
		t.Fatalf("new telemetry: %v", err)
	}

	_, span := telemetry.tracer.Start(context.Background(), "test span")
	span.End()
	if _, err := telemetry.LogWriter().
		Write([]byte(`{"level":"info","message":"hello"}` + "\n")); err != nil {
		t.Fatalf("write log: %v", err)
	}

	app := fiber.New()
	app.Use(telemetry.Middleware())
	app.Get("/ping", func(c fiber.Ctx) error { return c.SendString("pong") })
	if _, err := app.Test(httptest.NewRequest("GET", "/ping", nil)); err != nil {
		t.Fatalf("request through middleware: %v", err)
	}
	if err := app.Shutdown(); err != nil {
		t.Fatalf("shutdown app: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := telemetry.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	for _, path := range []string{
		"/api/kinstack/v1/traces",
		"/api/kinstack/v1/logs",
		"/api/kinstack/v1/metrics",
	} {
		auth, ok := paths[path]
		if !ok {
			t.Fatalf("expected export to %s, got %#v", path, paths)
		}
		if auth != "Basic dGVzdDp0ZXN0" {
			t.Fatalf("authorization for %s = %q", path, auth)
		}
	}
}

func TestTelemetryDisabledHasNoProviders(t *testing.T) {
	telemetry, err := New(Dependency{Config: config.Config{}})
	if err != nil {
		t.Fatalf("new telemetry: %v", err)
	}
	if telemetry.Enabled() {
		t.Fatal("telemetry enabled without configuration")
	}
	if telemetry.LogWriter() != nil {
		t.Fatal("log writer created without OpenObserve")
	}
	if err := telemetry.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
}
