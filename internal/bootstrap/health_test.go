package bootstrap

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"kinstack/config"

	"github.com/gofiber/fiber/v3"
)

func TestHealthDoesNotPublishEnvironment(t *testing.T) {
	app := fiber.New()
	app.Get("/health", healthHandler(config.Config{
		App: config.AppConfig{Name: "KinStack", Env: "production"},
	}))

	response, err := app.Test(httptest.NewRequest("GET", "/health", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	body, _ := io.ReadAll(response.Body)

	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status=%d", response.StatusCode)
	}
	text := string(body)
	if !strings.Contains(text, `"status":"ok"`) || !strings.Contains(text, "KinStack") {
		t.Fatalf("unexpected health payload: %s", text)
	}
	if strings.Contains(text, "env") || strings.Contains(text, "production") {
		t.Fatalf("health must not disclose the environment: %s", text)
	}
}
