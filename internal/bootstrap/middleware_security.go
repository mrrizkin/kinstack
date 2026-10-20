package bootstrap

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"kinstack/config"
	"kinstack/internal/pkg/debug"
	"kinstack/internal/pkg/log"
	"kinstack/internal/platform/security"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

// registerTransportMiddleware installs response headers, panic recovery,
// request ids, access logging, and telemetry. It runs before static assets so
// asset responses carry the same header and request-id contract without paying
// for a session load.
func registerTransportMiddleware(d GlobalMiddlewareDependency) {
	d.App.Use(helmet.New(helmet.Config{
		ContentTypeNosniff:        "nosniff",
		XFrameOptions:             "SAMEORIGIN",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionPolicy:          "camera=(self), microphone=(), geolocation=()",
		CrossOriginEmbedderPolicy: "unsafe-none",
		HSTSMaxAge:                hstsMaxAge(d.Config),
	}))
	d.App.Use(security.CSP(security.CSPConfig{
		Enabled:        d.Config.Security.CSPEnabled,
		ReportOnly:     d.Config.Security.CSPReportOnly,
		ReportURI:      d.Config.Security.CSPReportURI,
		ScriptSources:  cspSources(d.Config.Security.ExtraScriptSources, d.Config.Umami.ScriptURL),
		ConnectSources: cspSources(d.Config.Security.ExtraConnectSources, d.Config.GlitchTip.DSN, d.Config.Umami.ScriptURL),
		ImageSources:   cspSources(d.Config.Security.ExtraImageSources, d.Config.RustFS.PublicURL),
	}))
	d.App.Use(recoverer.New(recoverer.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c fiber.Ctx, e any) {
			if stackFrames, err := debug.StackTrace(); err == nil {
				c.Locals("recoverer_stack_trace", stackFrames)
			}
		},
	}))

	d.App.Use(requestid.New())
	d.App.Use(log.HTTPMiddleware(d.Log))
	d.App.Use(d.Telemetry.Middleware())

	// CSP reports arrive from browsers without a session or token, so the
	// endpoint is limited and answered before the session layer loads.
	cspLimiter := limiter.New(limiter.Config{
		Storage:      d.SharedStorage,
		Max:          cspReportRateLimitMax,
		Expiration:   time.Minute,
		KeyGenerator: func(c fiber.Ctx) string { return "csp:" + c.IP() },
		LimitReached: func(c fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) },
	})
	d.App.Use("/csp-report", cspLimiter)
}

// registerAccessMiddleware installs the session, rate limiting, bearer
// authentication, and CSRF protection that only application routes need.
func registerAccessMiddleware(d GlobalMiddlewareDependency) {
	d.App.Use(d.Session)

	// Bearer traffic is limited per token, not per session. The raw token is
	// hashed before it becomes a storage key; the limiter also covers malformed
	// or revoked tokens so credential probing cannot bypass the auth limiter.
	apiLimiter := limiter.New(limiter.Config{
		Storage:    d.SharedStorage,
		Max:        d.Config.App.APIRateLimitMax,
		Expiration: d.Config.App.APIRateLimitWindow,
		KeyGenerator: func(c fiber.Ctx) string {
			header := c.Get(fiber.HeaderAuthorization)
			sum := sha256.Sum256([]byte(header))
			return "api:" + hex.EncodeToString(sum[:])
		},
		Next: func(c fiber.Ctx) bool {
			return !strings.HasPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ")
		},
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "rate_limited",
					"message": "too many API requests",
				},
			})
		},
	})
	d.App.Use(apiLimiter)

	// Bearer tokens authenticate a request without a browser session. The
	// resolved actor is attached to the request only and never persisted.
	d.App.Use(func(c fiber.Ctx) error {
		header := c.Get(fiber.HeaderAuthorization)
		if !strings.HasPrefix(header, "Bearer ") {
			return c.Next()
		}
		raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		session, err := d.BearerAuth(c.Context(), raw)
		if err != nil || session == nil {
			return fiber.ErrUnauthorized
		}
		d.Actor.SetRequestActor(c, *session)
		return c.Next()
	})

	authLimiter := limiter.New(limiter.Config{
		Storage:    d.SharedStorage,
		Max:        d.Config.App.AuthRateLimitMax,
		Expiration: d.Config.App.AuthRateLimitWindow,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.IP() + ":" + c.Path()
		},
		Next: func(c fiber.Ctx) bool { return c.Method() != fiber.MethodPost },
	})
	d.App.Use("/login", authLimiter)
	d.App.Use("/2fa", authLimiter)
	registerCSRFMiddleware(d)
}

func hstsMaxAge(cfg config.Config) int {
	if cfg.App.Env == "production" {
		return 31536000
	}
	return 0
}

// cspReportRateLimitMax caps unauthenticated CSP reports per IP per minute.
const cspReportRateLimitMax = 60

// cspSources derives origins from absolute URLs and appends explicit extras.
func cspSources(extra []string, urls ...string) []string {
	sources := make([]string, 0, len(extra)+len(urls))
	for _, raw := range urls {
		if origin := security.Origin(raw); origin != "" {
			sources = append(sources, origin)
		}
	}
	return append(sources, extra...)
}

func registerCSRFMiddleware(d GlobalMiddlewareDependency) {
	// CSRF uses the server-side session store (synchronizer token pattern) so
	// the token is bound to one browser session and stays prefork-safe.
	d.App.Use(csrf.New(csrf.Config{
		Session:        d.SessionStore,
		CookieName:     "XSRF-TOKEN",
		CookiePath:     "/",
		CookieSameSite: "Lax",
		CookieSecure:   d.Config.App.Env == "production",
		Extractor:      extractors.FromHeader("X-XSRF-TOKEN"),
		CookieHTTPOnly: true,
		Next: func(c fiber.Ctx) bool {
			if strings.HasPrefix(c.Get(fiber.HeaderAuthorization), "Bearer ") {
				return true
			}
			// OAuth providers such as Apple post the authorization response back
			// as a form, and SAML uses HTTP-POST, so neither callback carries a
			// CSRF token. Both are protected by a single-use transaction record in
			// shared storage (sealed state for OAuth, random RelayState for SAML)
			// plus the provider's own response validation.
			return c.Path() == "/login/oauth/callback" || c.Path() == "/login/saml/acs" || c.Path() == "/csp-report"
		},
	}))
}
