package auth

import (
	"errors"
	"net/url"
	"strings"

	"kinstack/internal/platform/i18n"
	platformoauth "kinstack/internal/platform/oauth"
	platformsession "kinstack/internal/platform/session"

	"github.com/gofiber/fiber/v3"
)

const (
	// oauthVerifierFlashPrefix names the one-time session entry holding a PKCE
	// verifier between the authorization redirect and the callback.
	oauthVerifierFlashPrefix = "oauth_verifier:"
	// oauthContinuationFlashPrefix names session entries holding provider
	// continuation state (for example a marshaled goth session).
	oauthContinuationFlashPrefix = "oauth_continuation:"
)

func (h *Handler) oauthAvailable() bool {
	return !h.oauthRegistry.Empty() && h.oauthSigner != nil
}

// oauthContinuationStore backs provider continuation state with the Fiber
// session, which is available between the redirect and the callback.
type oauthContinuationStore struct{ c fiber.Ctx }

func (s oauthContinuationStore) Save(key, value string) error {
	return platformsession.SetFlash(s.c, oauthContinuationFlashPrefix+key, value)
}

func (s oauthContinuationStore) Load(key string) (string, bool) {
	return platformsession.ConsumeFlash(s.c, oauthContinuationFlashPrefix+key)
}

func (s oauthContinuationStore) Delete(key string) {
	_ = platformsession.SetFlash(s.c, oauthContinuationFlashPrefix+key, "")
}

// oauthStart begins the authorization-code flow for one registered provider.
func (h *Handler) oauthStart(c fiber.Ctx) error {
	if !h.oauthAvailable() {
		return fiber.ErrServiceUnavailable
	}
	provider, ok := h.oauthRegistry.Get(c.Query("provider"))
	if !ok {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	state, err := platformoauth.RandomToken(32)
	if err != nil {
		return h.internalServerError(c, "unable to start oauth login", err)
	}
	nonce, err := platformoauth.RandomToken(32)
	if err != nil {
		return h.internalServerError(c, "unable to start oauth login", err)
	}
	verifier, err := platformoauth.RandomToken(48)
	if err != nil {
		return h.internalServerError(c, "unable to start oauth login", err)
	}
	if err := platformsession.SetOAuthState(c, state, nonce, provider.Name()); err != nil {
		return fiber.ErrServiceUnavailable
	}
	if err := platformsession.SetFlash(c, oauthVerifierFlashPrefix+state, verifier); err != nil {
		return fiber.ErrServiceUnavailable
	}
	authURL, err := provider.Start(c.Context(), platformoauth.AuthorizationRequest{
		RedirectURI:  h.oauthRedirectURL,
		State:        state,
		Nonce:        nonce,
		CodeVerifier: verifier,
	}, oauthContinuationStore{c: c})
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(authURL)
}

// oauthCallback verifies the signed state, redeems the code, and signs the
// matching existing account in. It accepts POST because Apple posts the
// authorization response back as a form.
func (h *Handler) oauthCallback(c fiber.Ctx) error {
	if !h.oauthAvailable() {
		return fiber.ErrServiceUnavailable
	}
	state := firstNonEmpty(c.Query("state"), c.FormValue("state"))
	code := firstNonEmpty(c.Query("code"), c.FormValue("code"))
	requestedProvider := strings.TrimSpace(c.Query("provider"))

	nonce, providerName, matches := platformsession.ConsumeOAuthState(c, state)
	if !matches || code == "" {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	if requestedProvider != "" && requestedProvider != providerName {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	provider, ok := h.oauthRegistry.Get(providerName)
	if !ok {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	verifier, _ := platformsession.ConsumeFlash(c, oauthVerifierFlashPrefix+state)
	identity, err := provider.Complete(c.Context(), platformoauth.ExchangeRequest{
		Code:        code,
		Verifier:    verifier,
		Nonce:       nonce,
		State:       state,
		RedirectURI: h.oauthRedirectURL,
		Query: url.Values{
			"code":  {code},
			"state": {state},
		},
	}, oauthContinuationStore{c: c})
	if err != nil {
		if errors.Is(err, platformoauth.ErrUnavailable) {
			return fiber.ErrServiceUnavailable
		}
		return h.renderOAuthError(
			c,
			fiber.StatusUnprocessableEntity,
			i18n.T("auth.oauth_unverified"),
		)
	}
	session, err := h.service.OAuthSignIn(c.Context(), identity)
	if err != nil {
		var message string
		switch {
		case errors.Is(err, ErrOAuthUnregistered):
			message = i18n.T("auth.oauth_unregistered")
		case errors.Is(err, ErrOAuthEmailUnverified):
			message = i18n.T("auth.oauth_email_unverified")
		case errors.Is(err, ErrInactiveUser):
			message = i18n.T("auth.inactive_user")
		default:
			return h.internalServerError(c, "sign in with oauth", err)
		}
		return h.renderOAuthError(c, fiber.StatusForbidden, message)
	}
	return h.completeLogin(c, session)
}

func (h *Handler) renderOAuthError(c fiber.Ctx, status int, message string) error {
	props := h.loginProps()
	props["oauth_error"] = message
	return c.Status(status).Render("auth/LoginView", props)
}

func (h *Handler) internalServerError(c fiber.Ctx, message string, err error) error {
	h.logger.Errorw(message, "request_id", c.RequestID(), "error", err)
	return fiber.ErrInternalServerError
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
