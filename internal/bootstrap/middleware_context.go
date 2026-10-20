package bootstrap

import (
	"strings"

	platformactor "kinstack/internal/platform/actor"
	"kinstack/internal/platform/audit"
	"kinstack/internal/platform/authz"
	"kinstack/internal/platform/inertia"
	platformsession "kinstack/internal/platform/session"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/csrf"
)

// registerContextMiddleware installs audit context, actor/session validation,
// and the shared Inertia props consumed by every page.
func registerContextMiddleware(d GlobalMiddlewareDependency) {
	d.App.Use(audit.Middleware(d.Audit, d.Actor, func(err error) {
		d.Log.Errorw("audit event could not be recorded", "error", err)
	}))

	inertiaHandler := inertia.Middleware(d.Inertia)
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
