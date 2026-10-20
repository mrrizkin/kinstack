package auth

import (
	"errors"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"
	"kinstack/internal/platform/i18n"
	"kinstack/internal/platform/actor"
	platformoauth "kinstack/internal/platform/oauth"
	platformsaml "kinstack/internal/platform/saml"
	platformsession "kinstack/internal/platform/session"
	"kinstack/internal/platform/sessionregistry"
	"kinstack/internal/platform/twofactor"

	"github.com/gofiber/fiber/v3"
)

// HandlerDependency contains the service, actor boundary, session registry,
// two-factor service, and logger used by the handler.
type HandlerDependency struct {
	Service   *Service
	Actor     *actor.Actor
	Logger    log.Log
	Sessions  *sessionregistry.Service
	TwoFactor *twofactor.Service

	MagicLinkEnabled bool
	PasskeyEnabled   bool
	PasswordReset    bool

	OAuthProviders   []platformoauth.Provider
	OAuthStateSecret string
	OAuthRedirectURL string

	SAMLProvider *platformsaml.Provider
	SAMLACSURL   string
}

// Handler owns the HTTP boundary for authentication.
type Handler struct {
	service   *Service
	actor     *actor.Actor
	logger    log.Log
	sessions  *sessionregistry.Service
	twoFactor *twofactor.Service

	magicLinkEnabled bool
	passkeyEnabled   bool
	passwordReset    bool

	oauthRegistry    *platformoauth.Registry
	oauthSigner      *platformoauth.StateSigner
	oauthRedirectURL string

	samlProvider *platformsaml.Provider
	samlACSURL   string
}

// NewHandler constructs the auth HTTP handler.
func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "auth service is required")
	assert.Assert(d.Actor != nil, "auth actor boundary is required")
	assert.Assert(d.Logger != nil, "auth logger is required")

	handler := &Handler{
		service:   d.Service,
		actor:     d.Actor,
		logger:    d.Logger,
		sessions:  d.Sessions,
		twoFactor: d.TwoFactor,

		magicLinkEnabled: d.MagicLinkEnabled,
		passkeyEnabled:   d.PasskeyEnabled,
		passwordReset:    d.PasswordReset,

		samlProvider: d.SAMLProvider,
		samlACSURL:   d.SAMLACSURL,
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

// Mount registers the authentication routes at the root of the application.
func (h *Handler) Mount(router fiber.Router) {
	router.Get("/login", h.loginPage)
	router.Post("/login", h.login)
	if h.magicLinkEnabled {
		router.Post("/login/magic-link", h.requestMagicLink)
		router.Get("/login/magic-link/:token", h.consumeMagicLink)
	}
	if h.passwordReset {
		router.Get("/login/forgot", h.forgotPasswordPage)
		router.Post("/login/forgot", h.requestPasswordReset)
		router.Get("/login/reset/:token", h.resetPasswordPage)
		router.Post("/login/reset/:token", h.resetPassword)
	}
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
		h.logger.Errorw("render login page", "request_id", c.RequestID(), "error", err)
	}
	return err
}

// loginProps exposes the enabled sign-in methods to the sign-in page.
func (h *Handler) loginProps() fiber.Map {
	props := fiber.Map{
		"magic_link_enabled":     h.magicLinkEnabled,
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

func (h *Handler) login(c fiber.Ctx) error {
	var request LoginRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": i18n.T("common.invalid_payload"),
		})
	}

	session, err := h.service.Login(c.Context(), request.Username, request.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			props := h.loginProps()
			fieldErrors := fiber.Map{}
			if strings.TrimSpace(request.Username) == "" {
				fieldErrors["username"] = i18n.T("auth.username_required")
			}
			if request.Password == "" {
				fieldErrors["password"] = i18n.T("auth.password_required")
			}
			props["errors"] = fieldErrors
			return c.Status(fiber.StatusUnprocessableEntity).Render("auth/LoginView", props)
		case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInactiveUser):
			props := h.loginProps()
			props["errors"] = fiber.Map{"auth": err.Error()}
			return c.Status(fiber.StatusUnprocessableEntity).Render("auth/LoginView", props)
		case errors.Is(err, ErrAccountLocked):
			props := h.loginProps()
			props["errors"] = fiber.Map{"auth": err.Error()}
			return c.Status(fiber.StatusTooManyRequests).Render("auth/LoginView", props)
		default:
			h.logger.Errorw("login failed", "request_id", c.RequestID(), "error", err)
			return fiber.ErrInternalServerError
		}
	}

	if !session.Superuser && h.twoFactor != nil {
		enabled, err := h.twoFactor.IsEnabled(c.Context(), session.PublicID)
		if err != nil {
			h.logger.Errorw("check two-factor status", "request_id", c.RequestID(), "error", err)
			return fiber.ErrInternalServerError
		}
		if enabled {
			if err := platformsession.SetTwoFactorChallenge(c, session.PublicID); err != nil {
				h.logger.Errorw(
					"establish two-factor challenge",
					"request_id",
					c.RequestID(),
					"error",
					err,
				)
				return fiber.ErrInternalServerError
			}
			return c.Redirect().Status(fiber.StatusSeeOther).To(h.twoFactorPath())
		}
	}

	return h.establishSession(c, session)
}

// twoFactorPage renders the second-factor challenge for a valid password that
// still needs a TOTP or recovery code.
func (h *Handler) twoFactorPage(c fiber.Ctx) error {
	if _, ok := platformsession.PendingTwoFactorActor(c); !ok {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	if h.actor.ActorFromContext(c) != nil {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.homePath(false))
	}
	err := c.Render("auth/TwoFactorView", fiber.Map{})
	if err != nil {
		h.logger.Errorw("render two-factor page", "request_id", c.RequestID(), "error", err)
	}
	return err
}

// verifyTwoFactor completes a pending challenge and establishes the session.
func (h *Handler) verifyTwoFactor(c fiber.Ctx) error {
	publicID, ok := platformsession.PendingTwoFactorActor(c)
	if !ok {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	if h.twoFactor == nil {
		return fiber.ErrServiceUnavailable
	}
	var request TwoFactorRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": i18n.T("auth.two_factor_invalid"),
		})
	}
	verified, err := h.twoFactor.Verify(c.Context(), publicID, request.Code)
	if err != nil {
		h.logger.Errorw("verify two-factor code", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	if !verified {
		return c.Status(fiber.StatusUnprocessableEntity).Render("auth/TwoFactorView", fiber.Map{
			"errors": fiber.Map{"code": i18n.T("auth.invalid_code")},
		})
	}
	session, err := h.service.Session(c.Context(), publicID)
	if err != nil {
		h.logger.Errorw("load two-factor session", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	platformsession.ClearTwoFactorChallenge(c)
	return h.establishSession(c, session)
}

// requestMagicLink emails a single-use sign-in link. The response is identical
// whether or not the address belongs to an account.
func (h *Handler) requestMagicLink(c fiber.Ctx) error {
	var request MagicLinkRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": i18n.T("common.invalid_payload"),
		})
	}
	if err := h.service.RequestMagicLink(c.Context(), request.Email); err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return c.Status(fiber.StatusUnprocessableEntity).Render("auth/LoginView", fiber.Map{
				"errors": fiber.Map{"email": i18n.T("auth.email_invalid")},
			})
		}
		h.logger.Errorw("request magic link", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return c.Render("auth/LoginView", fiber.Map{
		"notice": i18n.T("auth.magic_link_sent"),
	})
}

// consumeMagicLink verifies a sign-in token and continues the normal sign-in
// flow (including the second factor when enabled).
func (h *Handler) consumeMagicLink(c fiber.Ctx) error {
	session, err := h.service.ConsumeMagicLink(c.Context(), c.Params("token"))
	if err != nil {
		if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrInvalidInput) {
			return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
		}
		h.logger.Errorw("consume magic link", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return h.completeLogin(c, session)
}

// forgotPasswordPage renders the self-service reset request form.
func (h *Handler) forgotPasswordPage(c fiber.Ctx) error {
	if !h.passwordReset {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	err := c.Render("auth/ForgotPasswordView", fiber.Map{})
	if err != nil {
		h.logger.Errorw("render forgot password page", "request_id", c.RequestID(), "error", err)
	}
	return err
}

// requestPasswordReset emails a reset link. The response is identical whether
// or not the address belongs to an account.
func (h *Handler) requestPasswordReset(c fiber.Ctx) error {
	if !h.passwordReset {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	var request ForgotPasswordRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).Render("auth/ForgotPasswordView", fiber.Map{
			"errors": fiber.Map{"email": i18n.T("common.invalid_payload")},
		})
	}
	if err := h.service.RequestPasswordReset(c.Context(), request.Email); err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return c.Status(fiber.StatusUnprocessableEntity).Render(
				"auth/ForgotPasswordView",
				fiber.Map{"errors": fiber.Map{"email": i18n.T("auth.email_invalid")}},
			)
		}
		h.logger.Errorw("request password reset", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return c.Render("auth/ForgotPasswordView", fiber.Map{
		"notice": i18n.T("auth.reset_link_sent"),
	})
}

// resetPasswordPage renders the new-password form for one reset token.
func (h *Handler) resetPasswordPage(c fiber.Ctx) error {
	if !h.passwordReset {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	err := c.Render("auth/ResetPasswordView", fiber.Map{"token": c.Params("token")})
	if err != nil {
		h.logger.Errorw("render reset password page", "request_id", c.RequestID(), "error", err)
	}
	return err
}

// resetPassword burns a reset token and stores the new password.
func (h *Handler) resetPassword(c fiber.Ctx) error {
	if !h.passwordReset {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	token := c.Params("token")
	var request ResetPasswordRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).Render("auth/ResetPasswordView", fiber.Map{
			"token":  token,
			"errors": fiber.Map{"new_password": i18n.T("common.invalid_payload")},
		})
	}
	if request.NewPassword != request.ConfirmPassword {
		return c.Status(fiber.StatusUnprocessableEntity).Render("auth/ResetPasswordView", fiber.Map{
			"token":  token,
			"errors": fiber.Map{"confirm_password": i18n.T("auth.password_mismatch")},
		})
	}
	if err := h.service.ResetPassword(c.Context(), token, request.NewPassword); err != nil {
		switch {
		case errors.Is(err, ErrInvalidToken):
			return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
		case errors.Is(err, ErrInvalidInput):
			return c.Status(fiber.StatusUnprocessableEntity).Render("auth/ResetPasswordView", fiber.Map{
				"token":  token,
				"errors": fiber.Map{"new_password": err.Error()},
			})
		default:
			h.logger.Errorw("reset password", "request_id", c.RequestID(), "error", err)
			return fiber.ErrInternalServerError
		}
	}
	return c.Render("auth/ResetPasswordView", fiber.Map{"done": true})
}

// completeLogin applies the shared second-factor challenge and then establishes
// the authenticated session.
func (h *Handler) completeLogin(c fiber.Ctx, session SessionResponse) error {
	if !session.Superuser && h.twoFactor != nil {
		enabled, err := h.twoFactor.IsEnabled(c.Context(), session.PublicID)
		if err != nil {
			h.logger.Errorw("check two-factor status", "request_id", c.RequestID(), "error", err)
			return fiber.ErrInternalServerError
		}
		if enabled {
			if err := platformsession.SetTwoFactorChallenge(c, session.PublicID); err != nil {
				h.logger.Errorw(
					"establish two-factor challenge",
					"request_id",
					c.RequestID(),
					"error",
					err,
				)
				return fiber.ErrInternalServerError
			}
			return c.Redirect().Status(fiber.StatusSeeOther).To(h.twoFactorPath())
		}
	}
	return h.establishSession(c, session)
}

// establishSession stores the authenticated actor, tracks the browser session,
// and redirects to the signed-in landing page.
func (h *Handler) establishSession(c fiber.Ctx, session SessionResponse) error {
	if err := h.setSession(c, session); err != nil {
		return err
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(h.homePath(session.Superuser))
}

// setSession stores the authenticated actor and tracks the browser session
// without redirecting. Non-password authenticators use it to finish sign-in.
func (h *Handler) setSession(c fiber.Ctx, session SessionResponse) error {
	if err := h.actor.SetActor(c, actor.SessionActor{
		PublicID:    session.PublicID,
		Username:    session.Username,
		DisplayName: session.DisplayName,
		Role:        session.Role,
		Superuser:   session.Superuser,
	}); err != nil {
		h.logger.Errorw("establish session", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	h.registerSession(c, session)
	return nil
}

// cancelTwoFactor abandons a pending challenge and returns to the login page.
func (h *Handler) cancelTwoFactor(c fiber.Ctx) error {
	platformsession.ClearTwoFactorChallenge(c)
	return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
}

// registerSession records the freshly authenticated browser session so it can
// be listed and revoked from the profile security page. The ephemeral
// break-glass super admin has no account and therefore no tracked session.
func (h *Handler) registerSession(c fiber.Ctx, session SessionResponse) {
	if h.sessions == nil || session.Superuser {
		return
	}
	if _, err := h.sessions.Register(c.Context(), sessionregistry.RegisterInput{
		UserPublicID: session.PublicID,
		SessionID:    platformsession.CurrentID(c),
		IP:           c.IP(),
		UserAgent:    c.Get(fiber.HeaderUserAgent),
	}); err != nil {
		h.logger.Errorw("register browser session", "request_id", c.RequestID(), "error", err)
	}
}

func (h *Handler) logout(c fiber.Ctx) error {
	sessionID := platformsession.CurrentID(c)
	if err := h.actor.ClearActor(c); err != nil {
		h.logger.Errorw("clear session", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	if h.sessions != nil {
		if err := h.sessions.RevokeSession(c.Context(), sessionID); err != nil {
			h.logger.Errorw("revoke browser session", "request_id", c.RequestID(), "error", err)
		}
	}
	// Root resolves to the auth path for signed-out visitors.
	return c.Redirect().Status(fiber.StatusSeeOther).To("/")
}
