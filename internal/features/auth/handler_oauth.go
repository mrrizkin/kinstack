package auth

import (
	"errors"
	"net/url"
	"strings"

	"kinstack/internal/pkg/log"

	"kinstack/internal/platform/i18n"
	platformoauth "kinstack/internal/platform/oauth"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) oauthAvailable() bool {
	return !h.oauthRegistry.Empty() && h.oauthSigner != nil && h.ssoStorage != nil
}

// ssoContinuation lives in shared storage, never in the session cookie, because
// POST callbacks are cross-site.
func (h *Handler) ssoContinuation() ssoStore {
	return newSSOStore(h.ssoStorage, ssoOAuthPrefix)
}

func (h *Handler) pkceVerifierStore() ssoStore {
	return newSSOStore(h.ssoStorage, ssoVerifierPrefix)
}

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
	}, h.now())
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

// oauthCallback opens the sealed state, redeems the code, and signs the matching account
// in. It accepts POST because providers such as Apple post the response back as a form.
// oauthReject logs a line naming why the callback failed, so a production login failure
// can be explained from the log.
func (h *Handler) oauthReject(c fiber.Ctx, reason string, cause error) error {
	if cause != nil {
		log.ForRequest(h.logger, c.RequestID()).
			Warnw("oauth callback rejected", "reason", reason, "error", cause)
	} else {
		log.ForRequest(h.logger, c.RequestID()).Warnw("oauth callback rejected", "reason", reason)
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
}

func (h *Handler) oauthCallback(c fiber.Ctx) error {
	if !h.oauthAvailable() {
		return fiber.ErrServiceUnavailable
	}
	rawState := firstNonEmpty(c.Query("state"), c.FormValue("state"))
	code := firstNonEmpty(c.Query("code"), c.FormValue("code"))
	requestedProvider := strings.TrimSpace(c.Query("provider"))

	payload, err := h.oauthSigner.Open(rawState, h.now())
	if err != nil || payload.Purpose != platformoauth.PurposeOAuth || code == "" {
		// The reason is logged, the value never is: a state blob is a bearer
		// of the transaction, and the redirect must not confirm which check
		// failed to the caller.
		return h.oauthReject(c, "state or code missing", err)
	}
	if requestedProvider != "" && requestedProvider != payload.Provider {
		return h.oauthReject(c, "provider mismatch", nil)
	}
	provider, ok := h.oauthRegistry.Get(payload.Provider)
	if !ok {
		return h.oauthReject(c, "provider not configured", nil)
	}
	verifier, verifierOK := h.pkceVerifierStore().Consume(rawState)
	if !verifierOK {
		// The verifier is the second half of PKCE. Without it the code
		// exchange would run unprotected, so the transaction stops here
		// rather than continue with an empty verifier.
		return h.oauthReject(c, "pkce verifier missing", nil)
	}
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
	log.ForRequest(h.logger, c.RequestID()).Errorw(message, "error", err)
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
