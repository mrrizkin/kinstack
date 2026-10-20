package auth

import (
	"bytes"
	"errors"
	"net/http"
	"strings"

	"kinstack/internal/platform/i18n"
	platformoauth "kinstack/internal/platform/oauth"
	platformsaml "kinstack/internal/platform/saml"
	platformsession "kinstack/internal/platform/session"

	"github.com/gofiber/fiber/v3"
)

const samlRequestIDFlashPrefix = "saml_request_id:"

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

// samlStart redirects the browser to the IdP with a signed AuthnRequest.
func (h *Handler) samlStart(c fiber.Ctx) error {
	if h.samlProvider == nil {
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
	if err := platformsession.SetFlash(c, samlRequestIDFlashPrefix+relayState, requestID); err != nil {
		return fiber.ErrServiceUnavailable
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(redirectURL)
}

// samlACS validates the IdP response and continues the normal sign-in flow.
func (h *Handler) samlACS(c fiber.Ctx) error {
	if h.samlProvider == nil {
		return fiber.ErrServiceUnavailable
	}
	relayState := strings.TrimSpace(c.FormValue("RelayState"))
	requestID, _ := platformsession.ConsumeFlash(c, samlRequestIDFlashPrefix+relayState)
	request, err := h.samlHTTPRequest(c)
	if err != nil {
		return h.internalServerError(c, "build saml callback request", err)
	}
	identity, err := h.samlProvider.Complete(request, requestID)
	if err != nil {
		if errors.Is(err, platformsaml.ErrUnavailable) {
			return fiber.ErrServiceUnavailable
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
