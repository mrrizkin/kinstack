package security

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestCSPDisabledIsNoOp(t *testing.T) {
	app := fiber.New()
	app.Use(CSP(CSPConfig{Enabled: false}))
	app.Get("/", func(c fiber.Ctx) error { return c.SendString(Nonce(c)) })

	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if got := resp.Header.Get("Content-Security-Policy"); got != "" {
		t.Fatalf("unexpected CSP header: %q", got)
	}
}

func TestCSPAddsNonceAndSources(t *testing.T) {
	app := fiber.New()
	app.Use(CSP(CSPConfig{
		Enabled:        true,
		ScriptSources:  []string{"https://umami.example.test", "https://umami.example.test"},
		ConnectSources: []string{"https://glitchtip.example.test"},
		ImageSources:   []string{"https://media.example.test"},
	}))
	app.Get("/", func(c fiber.Ctx) error { return c.SendString(Nonce(c)) })

	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	header := resp.Header.Get("Content-Security-Policy")
	if header == "" {
		t.Fatal("missing CSP header")
	}
	for _, fragment := range []string{
		"default-src 'self'",
		"script-src 'self' 'nonce-",
		"https://umami.example.test",
		"connect-src 'self' https://glitchtip.example.test",
		"img-src 'self' data: blob: https://media.example.test",
		"object-src 'none'",
	} {
		if !strings.Contains(header, fragment) {
			t.Fatalf("CSP %q missing %q", header, fragment)
		}
	}
	if strings.Count(header, "https://umami.example.test") != 1 {
		t.Fatalf("duplicate source not removed: %q", header)
	}
}

func TestCSPReportOnly(t *testing.T) {
	app := fiber.New()
	app.Use(CSP(CSPConfig{Enabled: true, ReportOnly: true, ReportURI: "/csp-report"}))
	app.Get("/", func(c fiber.Ctx) error { return c.SendString("ok") })

	resp, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.Header.Get("Content-Security-Policy-Report-Only") == "" {
		t.Fatal("missing report-only header")
	}
	if resp.Header.Get("Content-Security-Policy") != "" {
		t.Fatal("enforcing header set in report-only mode")
	}
	if !strings.Contains(
		resp.Header.Get("Content-Security-Policy-Report-Only"),
		"report-uri /csp-report",
	) {
		t.Fatal("missing report-uri directive")
	}
}

func TestOrigin(t *testing.T) {
	cases := map[string]string{
		"https://key@glitchtip.example.test/1": "https://glitchtip.example.test",
		"https://umami.example.test/script.js": "https://umami.example.test",
		"http://localhost:3000/script.js":      "http://localhost:3000",
		"/relative":                            "",
		"":                                     "",
	}
	for input, want := range cases {
		if got := Origin(input); got != want {
			t.Fatalf("Origin(%q)=%q, want %q", input, got, want)
		}
	}
}
