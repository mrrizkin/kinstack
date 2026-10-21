package profile

import (
	"kinstack/internal/platform/i18n"
	platforminertia "kinstack/internal/platform/inertia"
	platformsession "kinstack/internal/platform/session"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) updateProfile(c fiber.Ctx) error {
	var request UpdateProfileRequest
	if err := c.Bind().Body(&request); err != nil {
		return h.validationError(c, i18n.T("common.invalid_payload"), "form")
	}
	if _, err := h.service.UpdateDisplayName(c.Context(), h.actorPublicID(c), request); err != nil {
		return h.writeError(c, err, "display_name")
	}
	return platforminertia.RedirectWithFlash(
		h.inertia,
		c,
		h.profilePath(),
		i18n.T("profile.updated"),
	)
}

func (h *Handler) updatePreferences(c fiber.Ctx) error {
	var request UpdatePreferencesRequest
	if err := c.Bind().Body(&request); err != nil {
		return h.validationError(c, i18n.T("common.invalid_payload"), "form")
	}
	if _, err := h.service.UpdatePreferences(c.Context(), h.actorPublicID(c), request); err != nil {
		return h.writeError(c, err, "theme")
	}
	return platforminertia.RedirectWithFlash(
		h.inertia,
		c,
		h.profilePath(),
		i18n.T("profile.preferences_saved"),
	)
}

func (h *Handler) changePassword(c fiber.Ctx) error {
	var request ChangePasswordRequest
	if err := c.Bind().Body(&request); err != nil {
		return h.validationError(c, i18n.T("common.invalid_payload"), "form")
	}
	if err := h.service.ChangePassword(c.Context(), h.actorPublicID(c), request); err != nil {
		return h.writeError(c, err, passwordField(err))
	}
	// Roll the session identifier after a credential change.
	if sessionActor := h.actor.ActorFromContext(c); sessionActor != nil {
		if err := h.actor.SetActor(c, *sessionActor); err != nil {
			h.logger.Errorw("regenerate session after password change", "error", err)
		}
	}
	// A password change signs out every other device.
	if h.sessions != nil {
		if err := h.sessions.RevokeOthers(
			c.Context(),
			h.actorPublicID(c),
			platformsession.CurrentID(c),
		); err != nil {
			h.logger.Errorw("revoke other sessions after password change", "error", err)
		}
	}
	h.logger.Infow(
		"password changed",
		"request_id", c.RequestID(),
		"user", h.actorPublicID(c),
	)
	return platforminertia.RedirectWithFlash(
		h.inertia,
		c,
		h.profilePath(),
		i18n.T("profile.password_changed"),
	)
}
