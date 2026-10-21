package profile

import (
	"errors"

	"kinstack/internal/platform/i18n"
	platformsession "kinstack/internal/platform/session"
	"kinstack/internal/platform/sessionregistry"
	"kinstack/internal/platform/twofactor"

	"github.com/gofiber/fiber/v3"
)

// revokeSession signs out one other device.
func (h *Handler) revokeSession(c fiber.Ctx) error {
	if h.sessions == nil {
		return fiber.ErrServiceUnavailable
	}
	if err := h.sessions.Revoke(c.Context(), h.actorPublicID(c), c.Params("publicID")); err != nil {
		if errors.Is(err, sessionregistry.ErrNotFound) {
			return h.validationError(c, i18n.T("profile.session_not_found"), "form")
		}
		h.logger.Errorw("revoke session", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return h.inertia.RedirectWithFlash(c,
		h.profilePathFor(c),
		i18n.T("profile.session_revoked"),
	)
}

// revokeOtherSessions signs out every device except the current one.
func (h *Handler) revokeOtherSessions(c fiber.Ctx) error {
	if h.sessions == nil {
		return fiber.ErrServiceUnavailable
	}
	if err := h.sessions.RevokeOthers(
		c.Context(),
		h.actorPublicID(c),
		platformsession.CurrentID(c),
	); err != nil {
		h.logger.Errorw("revoke other sessions", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return h.inertia.RedirectWithFlash(c,
		h.profilePathFor(c),
		i18n.T("profile.other_sessions_revoked"),
	)
}

// beginTwoFactor stores a pending TOTP secret for the account.
func (h *Handler) beginTwoFactor(c fiber.Ctx) error {
	if h.twoFactor == nil {
		return fiber.ErrServiceUnavailable
	}
	account := h.actorPublicID(c)
	if sessionActor := h.actor.ActorFromContext(
		c,
	); sessionActor != nil &&
		sessionActor.Username != "" {
		account = sessionActor.Username
	}
	if _, err := h.twoFactor.BeginEnrollment(c.Context(), h.actorPublicID(c), account); err != nil {
		return h.writeTwoFactorError(c, err, "")
	}
	// The enrollment dialog shows the QR code, so no flash is needed.
	return c.Redirect().Status(fiber.StatusSeeOther).To(h.profilePathFor(c))
}

// confirmTwoFactor activates two-factor auth and stores the recovery codes for
// a single display.
func (h *Handler) confirmTwoFactor(c fiber.Ctx) error {
	if h.twoFactor == nil {
		return fiber.ErrServiceUnavailable
	}
	var request ConfirmTwoFactorRequest
	if err := c.Bind().Body(&request); err != nil {
		return h.validationError(c, i18n.T("common.invalid_payload"), "form")
	}
	codes, err := h.twoFactor.ConfirmEnrollment(c.Context(), h.actorPublicID(c), request.Code)
	if err != nil {
		return h.writeTwoFactorError(c, err, "code")
	}
	if err := platformsession.SetRecoveryCodes(c, codes); err != nil {
		h.logger.Errorw("store recovery codes", "request_id", c.RequestID(), "error", err)
	}
	return h.inertia.RedirectWithFlash(c,
		h.profilePathFor(c),
		i18n.T("profile.two_factor_enabled"),
	)
}

// disableTwoFactor turns off two-factor auth after a password check.
func (h *Handler) disableTwoFactor(c fiber.Ctx) error {
	if h.twoFactor == nil {
		return fiber.ErrServiceUnavailable
	}
	var request PasswordProtectedRequest
	if err := c.Bind().Body(&request); err != nil {
		return h.validationError(c, i18n.T("common.invalid_payload"), "form")
	}
	if err := h.service.VerifyCurrentPassword(
		c.Context(),
		h.actorPublicID(c),
		request.Password,
	); err != nil {
		return h.writeError(c, err, "two_factor_password")
	}
	if err := h.twoFactor.Disable(c.Context(), h.actorPublicID(c)); err != nil {
		return h.writeTwoFactorError(c, err, "")
	}
	return h.inertia.RedirectWithFlash(c,
		h.profilePathFor(c),
		i18n.T("profile.two_factor_disabled"),
	)
}

// regenerateRecoveryCodes issues a fresh set of one-time recovery codes.
func (h *Handler) regenerateRecoveryCodes(c fiber.Ctx) error {
	if h.twoFactor == nil {
		return fiber.ErrServiceUnavailable
	}
	var request PasswordProtectedRequest
	if err := c.Bind().Body(&request); err != nil {
		return h.validationError(c, i18n.T("common.invalid_payload"), "form")
	}
	if err := h.service.VerifyCurrentPassword(
		c.Context(),
		h.actorPublicID(c),
		request.Password,
	); err != nil {
		return h.writeError(c, err, "two_factor_password")
	}
	codes, err := h.twoFactor.RegenerateRecoveryCodes(c.Context(), h.actorPublicID(c))
	if err != nil {
		return h.writeTwoFactorError(c, err, "")
	}
	if err := platformsession.SetRecoveryCodes(c, codes); err != nil {
		h.logger.Errorw("store recovery codes", "request_id", c.RequestID(), "error", err)
	}
	return h.inertia.RedirectWithFlash(c,
		h.profilePathFor(c),
		i18n.T("profile.recovery_codes_created"),
	)
}

// writeTwoFactorError maps two-factor sentinel errors onto form fields.
func (h *Handler) writeTwoFactorError(c fiber.Ctx, err error, field string) error {
	switch {
	case errors.Is(err, twofactor.ErrInvalidCode):
		return h.validationError(c, i18n.T("profile.invalid_code"), "code")
	case errors.Is(err, twofactor.ErrNotFound):
		return h.inertia.FailedAction(c,
			h.profilePathFor(c),
			fiber.StatusNotFound,
			err.Error(),
		)
	case errors.Is(err, twofactor.ErrInvalidInput),
		errors.Is(err, twofactor.ErrNotEnabled),
		errors.Is(err, twofactor.ErrAlreadyEnabled),
		errors.Is(err, twofactor.ErrNotConfigured):
		return h.validationError(c, err.Error(), field)
	default:
		h.logger.Errorw("two-factor request failed", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
}
