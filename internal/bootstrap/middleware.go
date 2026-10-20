package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"kinstack/config"
	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/debug"
	"kinstack/internal/pkg/log"
	platformactor "kinstack/internal/platform/actor"
	"kinstack/internal/platform/audit"
	"kinstack/internal/platform/authz"
	"kinstack/internal/platform/inertia"
	"kinstack/internal/platform/observability"
	"kinstack/internal/platform/pdf"
	"kinstack/internal/platform/security"
	platformsession "kinstack/internal/platform/session"
	"kinstack/internal/platform/sessionregistry"

	"codeberg.org/mrrizkin/fibertia"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/extractors"
	"github.com/gofiber/fiber/v3/middleware/csrf"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	fibersession "github.com/gofiber/fiber/v3/middleware/session"
)

// GlobalMiddlewareDependency contains the boundaries used by global middleware.
type GlobalMiddlewareDependency struct {
	App           *fiber.App
	Config        config.Config
	Inertia       *fibertia.Fibertia
	Session       fiber.Handler
	SessionStore  *fibersession.Store
	SharedStorage fiber.Storage
	Actor         *platformactor.Actor
	Authz         *authz.Service
	Audit         audit.Recorder
	Log           *log.Logger
	Documents     *pdf.Client
	Sessions      *sessionregistry.Service
	BearerAuth    func(context.Context, string) (*platformactor.SessionActor, error)
	Telemetry     *observability.Telemetry
}

// RegisterGlobalMiddleware wires security, logging, session, audit, CSRF, and
// Inertia middleware in the correct order.
func RegisterGlobalMiddleware(d GlobalMiddlewareDependency) {
	assert.Assert(d.App != nil, "app is required")
	assert.Assert(d.Inertia != nil, "inertia is required")
	assert.Assert(d.Session != nil, "session middleware is required")
	assert.Assert(d.SessionStore != nil, "session store is required")
	assert.Assert(d.Actor != nil, "actor is required")
	assert.Assert(d.Log != nil, "logger is required")
	assert.Assert(d.Documents != nil, "document renderer is required")
	assert.Assert(d.BearerAuth != nil, "bearer authenticator is required")

	inertiaHandler := inertia.Middleware(d.Inertia)

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
					"message": "terlalu banyak permintaan API",
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

	d.App.Use(audit.Middleware(d.Audit, d.Actor, func(err error) {
		d.Log.Errorw("audit event could not be recorded", "error", err)
	}))

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
			// as a form. The signed state protects the endpoint instead of CSRF.
			return c.Path() == "/login/oauth/callback" || c.Path() == "/login/saml/acs" || c.Path() == "/csp-report"
		},
	}))

	d.App.Use(func(c fiber.Ctx) error {
		if skipStatic(c) {
			return c.Next()
		}
		return inertiaHandler(c)
	})
	d.App.Use(func(c fiber.Ctx) error {
		if skipStatic(c) {
			return c.Next()
		}
		d.Inertia.ShareWith(c, "app_name", d.Config.App.Name)
		d.Inertia.ShareWith(c, "csrf_token", csrf.TokenFromContext(c))
		d.Inertia.ShareWith(c, "pdf_enabled", d.Documents.Enabled())
		d.Inertia.ShareWith(c, "auth_methods", fiber.Map{
			"magic_link": d.Config.Auth.MagicLink.Enabled,
			"passkey":    d.Config.Auth.Passkey.Enabled,
			"api_token":  d.Config.Auth.APIToken.Enabled,
			"oauth":      d.Config.Auth.OAuthEnabled(),
		})
		actor := d.Actor.ActorFromContext(c)
		if actor != nil && d.Actor.ValidateSession(c.Context(), actor) != nil {
			_ = d.Actor.ClearActor(c)
			actor = nil
		}
		d.Inertia.ShareWith(c, "actor", actor)
		d.Inertia.ShareWith(c, "authz_permissions", effectivePermissions(c, d.Authz, actor))
		d.Inertia.ShareWith(c, "observability", fiber.Map{
			"glitchtip": fiber.Map{
				"dsn":                d.Config.GlitchTip.DSN,
				"environment":        d.Config.GlitchTip.Environment,
				"release":            d.Config.GlitchTip.Release,
				"traces_sample_rate": d.Config.GlitchTip.TracesSampleRate,
			},
		})
		d.Inertia.ShareWith(c, "analytics", fiber.Map{
			"umami": fiber.Map{
				"enabled":    d.Config.Umami.Enabled,
				"script_url": d.Config.Umami.ScriptURL,
				"website_id": d.Config.Umami.WebsiteID,
				"domains":    d.Config.Umami.Domains,
				"auto_track": d.Config.Umami.AutoTrack,
			},
		})
		if actor != nil && !actor.Superuser && d.Sessions != nil {
			if err := d.Sessions.Touch(c.Context(), platformsession.CurrentID(c)); err != nil {
				d.Log.Errorw("touch browser session", "request_id", c.RequestID(), "error", err)
			}
		}
		return c.Next()
	})
	d.Log.Infow("Middleware configured")
}

func skipStatic(c fiber.Ctx) bool {
	path := c.Path()
	return strings.HasPrefix(path, "/build/") || strings.HasPrefix(path, "/uploads/")
}

// effectivePermissions resolves the actor's permission codes for the frontend.
// A resolution failure degrades to an empty list rather than blocking render.
func effectivePermissions(
	c fiber.Ctx,
	service *authz.Service,
	session *platformactor.SessionActor,
) []string {
	permissions := make([]string, 0)
	if service == nil || session == nil {
		return permissions
	}
	codes, err := service.Permissions(c.Context(), authz.PrincipalFromActor(session))
	if err != nil {
		return permissions
	}
	for _, code := range codes {
		permissions = append(permissions, string(code))
	}
	return permissions
}

func hstsMaxAge(cfg config.Config) int {
	if cfg.App.Env == "production" {
		return 31536000
	}
	return 0
}

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
