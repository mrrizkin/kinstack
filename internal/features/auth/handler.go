package auth

import (
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/clock"
	platformoauth "kinstack/internal/platform/oauth"
	platformsaml "kinstack/internal/platform/saml"
	"kinstack/internal/platform/sessionregistry"
	"kinstack/internal/platform/twofactor"

	"github.com/gofiber/fiber/v3"
)

type HandlerDependency struct {
	Service   *Service
	Actor     *actor.Actor
	Logger    log.Log
	Sessions  *sessionregistry.Service
	TwoFactor *twofactor.Service

	PasskeyEnabled bool
	PasswordReset  bool

	// Now supplies the wall clock for signed OAuth state. Nil falls back to
	// the system clock.
	Now func() time.Time

	OAuthProviders   []platformoauth.Provider
	OAuthStateSecret string
	OAuthRedirectURL string

	SAMLProvider *platformsaml.Provider
	SAMLACSURL   string

	// ContinuationStorage is the shared server-side storage that carries SSO
	// transaction data across the cross-site callback. It is required whenever
	// OAuth or SAML is enabled; a browser session cookie cannot be relied on
	// because POST callbacks do not send SameSite=Lax cookies.
	ContinuationStorage fiber.Storage
}

type Handler struct {
	service   *Service
	actor     *actor.Actor
	logger    log.Log
	sessions  *sessionregistry.Service
	twoFactor *twofactor.Service

	passkeyEnabled bool
	passwordReset  bool

	oauthRegistry    *platformoauth.Registry
	oauthSigner      *platformoauth.StateSigner
	oauthRedirectURL string

	samlProvider *platformsaml.Provider
	samlACSURL   string

	ssoStorage fiber.Storage
	now        func() time.Time
}

func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "auth service is required")
	assert.Assert(d.Actor != nil, "auth actor boundary is required")
	assert.Assert(d.Logger != nil, "auth logger is required")

	now := d.Now
	if now == nil {
		now = clock.System
	}

	handler := &Handler{
		service:   d.Service,
		actor:     d.Actor,
		logger:    d.Logger,
		sessions:  d.Sessions,
		twoFactor: d.TwoFactor,

		passkeyEnabled: d.PasskeyEnabled,
		passwordReset:  d.PasswordReset,

		samlProvider: d.SAMLProvider,
		samlACSURL:   d.SAMLACSURL,

		ssoStorage: d.ContinuationStorage,
		now:        now,
	}
	if len(d.OAuthProviders) > 0 || d.SAMLProvider != nil {
		assert.Assert(d.ContinuationStorage != nil, "sso continuation storage is required")
	}
	if len(d.OAuthProviders) > 0 {
		assert.Assert(d.OAuthStateSecret != "", "oauth state secret is required")
		signer, err := platformoauth.NewStateSigner(d.OAuthStateSecret, 10*time.Minute)
		assert.NoError(err)
		registry := platformoauth.NewRegistry()
		for _, provider := range d.OAuthProviders {
			registry.Register(provider)
		}
		handler.oauthRegistry = registry
		handler.oauthSigner = signer
		handler.oauthRedirectURL = d.OAuthRedirectURL
	}
	return handler
}

func (h *Handler) Mount(router fiber.Router) {
	router.Get("/login", h.loginPage)
	router.Post("/login", h.login)
	if h.passwordReset {
		router.Get("/login/forgot", h.forgotPasswordPage)
		router.Post("/login/forgot", h.requestPasswordReset)
	}
	// Reset routes stay mounted when self-service reset is off: an administrator
	// invitation delivers the same single-use token to activate a new account.
	router.Get("/login/reset/:token", h.resetPasswordPage)
	router.Post("/login/reset/:token", h.resetPassword)
	router.Get("/login/oauth/start", h.oauthStart)
	router.Get("/login/oauth/callback", h.oauthCallback)
	router.Post("/login/oauth/callback", h.oauthCallback)
	if h.samlProvider != nil {
		router.Get("/saml/metadata", h.samlMetadata)
		router.Get("/login/saml/start", h.samlStart)
		router.Post("/login/saml/acs", h.samlACS)
	}
	router.Get("/2fa", h.twoFactorPage)
	router.Post("/2fa", h.verifyTwoFactor)
	router.Post("/2fa/cancel", h.cancelTwoFactor)
	router.Post("/logout", h.logout)
}

func (h *Handler) loginPath() string {
	return "/login"
}

func (h *Handler) twoFactorPath() string {
	return "/2fa"
}

// homePath returns the landing route after sign-in. Every actor lands on the
// dashboard; account-scoped pages are available from the account menu.
func (h *Handler) homePath(_ bool) string {
	return "/dashboard"
}

func (h *Handler) loginPage(c fiber.Ctx) error {
	if session := h.actor.ActorFromContext(c); session != nil {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.homePath(session.Superuser))
	}
	err := c.Render("auth/LoginView", h.loginProps())
	if err != nil {
		log.ForRequest(h.logger, c.RequestID()).Errorw("render login page", "error", err)
	}
	return err
}

// loginProps exposes the enabled sign-in methods to the sign-in page.
func (h *Handler) loginProps() fiber.Map {
	props := fiber.Map{
		"passkey_enabled":        h.passkeyEnabled,
		"password_reset_enabled": h.passwordReset,
	}
	if h.oauthAvailable() {
		providers := make([]fiber.Map, 0)
		for _, provider := range h.oauthRegistry.Providers() {
			providers = append(providers, fiber.Map{
				"name":  provider.Name(),
				"label": provider.Label(),
			})
		}
		props["oauth_enabled"] = true
		props["oauth_providers"] = providers
	}
	props["saml_enabled"] = h.samlProvider != nil
	return props
}
