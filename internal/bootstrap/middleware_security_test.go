package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"kinstack/config"
	"kinstack/internal/pkg/log"

	"github.com/gofiber/fiber/v3"
	fibersession "github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/storage/memory/v2"
)

// The transport and CSRF layers are assembled directly: NewApp also needs a
// database and every feature slice, which these assertions do not exercise.
func newSecurityStack(t *testing.T, env string) *fiber.App {
	t.Helper()

	cfg := config.Config{
		App:      config.AppConfig{Name: "KinStack", Env: env, URL: "http://localhost:3000"},
		Security: config.SecurityConfig{CSPEnabled: true},
		Log: config.LogConfig{
			Level:     "error",
			File:      true,
			Dir:       t.TempDir(),
			Name:      "test",
			MaxBackup: 1,
			MaxSize:   1,
			MaxAge:    1,
		},
	}
	store := fibersession.NewStore(fibersession.Config{
		Storage:         memory.New(),
		IdleTimeout:     30 * time.Second,
		AbsoluteTimeout: time.Minute,
	})
	app := fiber.New()
	dependency := GlobalMiddlewareDependency{
		App:          app,
		Config:       cfg,
		Log:          log.NewLogger(log.LoggerDependency{Config: cfg}),
		SessionStore: store,
	}
	registerTransportMiddleware(dependency)
	registerCSRFMiddleware(dependency)

	ok := func(c fiber.Ctx) error { return c.SendString("ok") }
	app.Get("/page", ok)
	app.Post("/page", ok)
	app.Post("/login/oauth/callback", ok)
	app.Post("/login/saml/acs", ok)
	app.Post("/csp-report", ok)
	return app
}

func doRequest(
	t *testing.T,
	app *fiber.App,
	method, path string,
	headers map[string]string,
) *http.Response {
	t.Helper()

	request := httptest.NewRequest(method, path, nil)
	request.Host = "localhost:3000"
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := app.Test(request, fiber.TestConfig{Timeout: 5 * time.Second})
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func TestTransportMiddlewareSetsSecurityHeaders(t *testing.T) {
	app := newSecurityStack(t, "production")
	response := doRequest(t, app, fiber.MethodGet, "/page", nil)

	headers := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "SAMEORIGIN",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}
	for name, want := range headers {
		if got := response.Header.Get(name); got != want {
			t.Fatalf("%s = %q, want %q", name, got, want)
		}
	}
	if response.Header.Get("X-Request-Id") == "" {
		t.Fatal("request id header is missing")
	}
	policy := response.Header.Get("Content-Security-Policy")
	if !strings.Contains(policy, "default-src 'self'") || !strings.Contains(policy, "'nonce-") {
		t.Fatalf("unexpected CSP: %q", policy)
	}
}

func TestHSTSMaxAgeIsProductionOnly(t *testing.T) {
	production := config.Config{App: config.AppConfig{Env: "production"}}
	if got := hstsMaxAge(production); got != 31536000 {
		t.Fatalf("production HSTS = %d", got)
	}
	development := config.Config{App: config.AppConfig{Env: "development"}}
	if got := hstsMaxAge(development); got != 0 {
		t.Fatalf("development HSTS = %d", got)
	}
}

func TestCSRFRejectsPostWithoutToken(t *testing.T) {
	app := newSecurityStack(t, "production")
	response := doRequest(t, app, fiber.MethodPost, "/page", nil)

	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusForbidden)
	}
}

func TestCSRFAcceptsTheCookieTokenInTheHeader(t *testing.T) {
	app := newSecurityStack(t, "production")
	page := doRequest(t, app, fiber.MethodGet, "/page", nil)

	var token string
	pairs := make([]string, 0, len(page.Cookies()))
	for _, cookie := range page.Cookies() {
		pairs = append(pairs, cookie.Name+"="+cookie.Value)
		if cookie.Name == "XSRF-TOKEN" {
			token = cookie.Value
		}
	}
	if token == "" {
		t.Fatal("GET did not issue a CSRF cookie")
	}

	response := doRequest(t, app, fiber.MethodPost, "/page", map[string]string{
		"Cookie":         strings.Join(pairs, "; "),
		"X-XSRF-TOKEN":   token,
		"Sec-Fetch-Site": "same-origin",
	})
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, fiber.StatusOK)
	}
}

func TestCSRFExemptsBearerAndProviderCallbacks(t *testing.T) {
	app := newSecurityStack(t, "production")

	cases := map[string]struct {
		path    string
		headers map[string]string
	}{
		"bearer": {
			path:    "/page",
			headers: map[string]string{"Authorization": "Bearer token"},
		},
		"oauth callback": {path: "/login/oauth/callback"},
		"saml acs":       {path: "/login/saml/acs"},
		"csp report":     {path: "/csp-report"},
	}
	for name, test := range cases {
		response := doRequest(t, app, fiber.MethodPost, test.path, test.headers)
		if response.StatusCode != fiber.StatusOK {
			t.Fatalf("%s: status = %d, want %d", name, response.StatusCode, fiber.StatusOK)
		}
	}
}
