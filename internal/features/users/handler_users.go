package users

import (
	"kinstack/internal/platform/i18n"
	platforminertia "kinstack/internal/platform/inertia"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) create(c fiber.Ctx) error {
	var request CreateUserRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": i18n.T("common.invalid_payload")})
	}
	if _, err := h.service.Create(c.Context(), request); err != nil {
		return h.writeError(c, err)
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.usersPath(), i18n.T("users.created"))
}

func (h *Handler) invite(c fiber.Ctx) error {
	var request InviteUserRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": i18n.T("common.invalid_payload")})
	}
	if _, err := h.service.Invite(c.Context(), request); err != nil {
		return h.writeError(c, err)
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.usersPath(), i18n.T("auth.invitation_sent"))
}

func (h *Handler) update(c fiber.Ctx) error {
	var request UpdateUserRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": i18n.T("common.invalid_payload")})
	}
	if _, err := h.service.Update(c.Context(), c.Params("publicID"), request); err != nil {
		return h.writeError(c, err)
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.usersPath(), i18n.T("users.updated"))
}

func (h *Handler) resetPassword(c fiber.Ctx) error {
	var request ResetPasswordRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": i18n.T("common.invalid_payload")})
	}
	if err := h.service.ResetPassword(
		c.Context(),
		c.Params("publicID"),
		request.Password,
	); err != nil {
		return h.writeError(c, err)
	}
	return platforminertia.RedirectWithFlash(
		h.inertia,
		c,
		h.usersPath(),
		i18n.T("users.password_updated"),
	)
}

func (h *Handler) setActive(c fiber.Ctx) error {
	var request SetActiveRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": i18n.T("common.invalid_payload")})
	}
	if _, err := h.service.SetActive(
		c.Context(),
		c.Params("publicID"),
		request.IsActive,
	); err != nil {
		return h.writeError(c, err)
	}
	return platforminertia.RedirectWithFlash(
		h.inertia,
		c,
		h.usersPath(),
		i18n.T("users.status_updated"),
	)
}
