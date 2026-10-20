package auth

import (
	"bytes"
	"errors"
	"net/http"
	"strings"

	"kinstack/internal/platform/i18n"
	platformoauth "kinstack/internal/platform/oauth"
	platformsaml "kinstack/internal/platform/saml"

	"github.com/gofiber/fiber/v3"
)

// samlMetadata serves the service-provider metadata for IdP registration.
func (h *Handler) samlMetadata(c fiber.Ctx) error {
	if h.samlProvider == nil {
		return fiber.ErrServiceUnavailable
	}
	metadata, err := h.samlProvider.Metadata()
	if err != nil {
		h.logger.Errorw("render saml metadata", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	c.Set(fiber.HeaderContentType, "application/samlmetadata+xml")
	return c.Send(metadata)
}

// samlStart redirects the browser to the IdP with an AuthnRequest. The
// request ID is kept in shared storage keyed by a random RelayState instead of
// the session: an HTTP-POST ACS callback is cross-site and carries no
// SameSite=Lax cookie.
func (h *Handler) samlStart(c fiber.Ctx) error {
	if h.samlProvider == nil || h.ssoStorage == nil {
		return fiber.ErrServiceUnavailable
	}
	relayState, err := platformoauth.RandomToken(32)
	if err != nil {
		return h.internalServerError(c, "unable to start saml login", err)
	}
	redirectURL, requestID, err := h.samlProvider.Start(relayState)
	if err != nil {
		return fiber.ErrServiceUnavailable
	}
	if err := h.samlRequestIDStore().Save(relayState, requestID); err != nil {
		return fiber.ErrServiceUnavailable
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(redirectURL)
}

// samlRequestIDStore holds the AuthnRequest ID of one SAML transaction.
func (h *Handler) samlRequestIDStore() ssoStore {
	return newSSOStore(h.ssoStorage, ssoSAMLRequestID)
}

// samlACS validates the IdP response and continues the normal sign-in flow.
func (h *Handler) samlACS(c fiber.Ctx) error {
	if h.samlProvider == nil || h.ssoStorage == nil {
		return fiber.ErrServiceUnavailable
	}
	relayState := strings.TrimSpace(c.FormValue("RelayState"))
	requestID, ok := h.samlRequestIDStore().Consume(relayState)
	if relayState == "" || !ok {
		// Without the transaction record the InResponseTo check cannot run and
		// the assertion must not be trusted (fail closed).
		return h.renderOAuthError(
			c,
			fiber.StatusUnprocessableEntity,
			i18n.T("auth.saml_unverified"),
		)
	}
	request, err := h.samlHTTPRequest(c)
	if err != nil {
		return h.internalServerError(c, "build saml callback request", err)
	}
	identity, err := h.samlProvider.Complete(request, requestID)
	if err != nil {
		if errors.Is(err, platformsaml.ErrUnavailable) {
			return fiber.ErrServiceUnavailable
		}
		if errors.Is(err, platformsaml.ErrUnstableSubject) {
			return h.renderOAuthError(
				c,
				fiber.StatusUnprocessableEntity,
				i18n.T("auth.saml_unstable_subject"),
			)
		}
		return h.renderOAuthError(
			c,
			fiber.StatusUnprocessableEntity,
			i18n.T("auth.saml_unverified"),
		)
	}
	session, err := h.service.SAMLSignIn(c.Context(), identity)
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
			return h.internalServerError(c, "sign in with saml", err)
		}
		return h.renderOAuthError(c, fiber.StatusForbidden, message)
	}
	return h.completeLogin(c, session)
}

// samlHTTPRequest rebuilds the callback as a standard library request so the
// SAML library can validate the posted assertion.
func (h *Handler) samlHTTPRequest(c fiber.Ctx) (*http.Request, error) {
	request, err := http.NewRequestWithContext(
		c.Context(),
		http.MethodPost,
		h.samlACSURL,
		bytes.NewReader(c.Body()),
	)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", c.Get(fiber.HeaderContentType))
	if err := request.ParseForm(); err != nil {
		return nil, err
	}
	return request, nil
}
