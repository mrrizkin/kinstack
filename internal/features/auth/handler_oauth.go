package auth

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"kinstack/internal/platform/i18n"
	platformoauth "kinstack/internal/platform/oauth"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) oauthAvailable() bool {
	return !h.oauthRegistry.Empty() && h.oauthSigner != nil && h.ssoStorage != nil
}

// ssoContinuation returns the namespaced store that carries provider
// continuation blobs (for example a marshaled goth session) across the
// callback. It lives in shared server-side storage, never in the session
// cookie, because POST callbacks are cross-site.
func (h *Handler) ssoContinuation() ssoStore {
	return newSSOStore(h.ssoStorage, ssoOAuthPrefix)
}

// pkceVerifierStore holds the PKCE verifier of one authorization request.
func (h *Handler) pkceVerifierStore() ssoStore {
	return newSSOStore(h.ssoStorage, ssoVerifierPrefix)
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
	sealed, err := h.oauthSigner.Seal(platformoauth.StatePayload{
		Purpose:  platformoauth.PurposeOAuth,
		State:    state,
		Provider: provider.Name(),
		Nonce:    nonce,
	}, time.Now())
	if err != nil {
		return h.internalServerError(c, "unable to seal oauth state", err)
	}
	// The verifier is keyed by the sealed state so a callback can only redeem
	// the request it was issued for, and only once.
	if err := h.pkceVerifierStore().Save(sealed, verifier); err != nil {
		return fiber.ErrServiceUnavailable
	}
	authURL, err := provider.Start(c.Context(), platformoauth.AuthorizationRequest{
		RedirectURI:  h.oauthRedirectURL,
		State:        sealed,
		Nonce:        nonce,
		CodeVerifier: verifier,
	}, h.ssoContinuation())
	if err != nil {
		h.pkceVerifierStore().Delete(sealed)
		return fiber.ErrServiceUnavailable
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(authURL)
}

// oauthCallback opens the sealed state, redeems the code, and signs the
// matching existing account in. It accepts POST because providers such as Apple
// post the authorization response back as a form. No session cookie is
// required: the state is sealed, and transaction data lives in shared storage.
func (h *Handler) oauthCallback(c fiber.Ctx) error {
	if !h.oauthAvailable() {
		return fiber.ErrServiceUnavailable
	}
	rawState := firstNonEmpty(c.Query("state"), c.FormValue("state"))
	code := firstNonEmpty(c.Query("code"), c.FormValue("code"))
	requestedProvider := strings.TrimSpace(c.Query("provider"))

	payload, err := h.oauthSigner.Open(rawState, time.Now())
	if err != nil || payload.Purpose != platformoauth.PurposeOAuth || code == "" {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	if requestedProvider != "" && requestedProvider != payload.Provider {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	provider, ok := h.oauthRegistry.Get(payload.Provider)
	if !ok {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	verifier, _ := h.pkceVerifierStore().Consume(rawState)
	// The provider continuation blob is single use as well.
	defer h.ssoContinuation().Delete(rawState)
	identity, err := provider.Complete(c.Context(), platformoauth.ExchangeRequest{
		Code:        code,
		Verifier:    verifier,
		Nonce:       payload.Nonce,
		State:       rawState,
		RedirectURI: h.oauthRedirectURL,
		Query: url.Values{
			"code":  {code},
			"state": {rawState},
		},
	}, h.ssoContinuation())
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
