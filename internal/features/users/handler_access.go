package users

import (
	"kinstack/internal/platform/i18n"
	platforminertia "kinstack/internal/platform/inertia"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) createRole(c fiber.Ctx) error {
	var request CreateRoleRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": i18n.T("common.invalid_payload")})
	}
	if _, err := h.service.CreateRole(c.Context(), request); err != nil {
		return h.writeError(c, err)
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.accessPath(), i18n.T("users.role_created"))
}

func (h *Handler) updateRole(c fiber.Ctx) error {
	var request UpdateRoleRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": i18n.T("common.invalid_payload")})
	}
	if _, err := h.service.UpdateRole(c.Context(), c.Params("code"), request); err != nil {
		return h.writeError(c, err)
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.accessPath(), i18n.T("users.role_updated_flash"))
}

func (h *Handler) deleteRole(c fiber.Ctx) error {
	if err := h.service.DeleteRole(c.Context(), c.Params("code")); err != nil {
		return h.writeError(c, err)
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.accessPath(), i18n.T("users.role_deleted"))
}

func (h *Handler) setRolePermissions(c fiber.Ctx) error {
	var request SetRolePermissionsRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": i18n.T("common.invalid_payload")})
	}
	if err := h.service.SetRolePermissions(
		c.Context(),
		c.Params("code"),
		request.Permissions,
	); err != nil {
		return h.writeError(c, err)
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.accessPath(), i18n.T("users.permissions_updated"))
}

func (h *Handler) assignRole(c fiber.Ctx) error {
	var request AssignRoleRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": i18n.T("common.invalid_payload")})
	}
	if err := h.service.AssignRole(c.Context(), c.Params("publicID"), request.Role); err != nil {
		return h.writeError(c, err)
	}
	return platforminertia.RedirectWithFlash(
		h.inertia,
		c,
		h.accessPath(),
		i18n.T("users.role_updated"),
	)
}

func (h *Handler) setOverride(c fiber.Ctx) error {
	var request SetOverrideRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": i18n.T("common.invalid_payload")})
	}
	if err := h.service.SetOverride(c.Context(), c.Params("publicID"), request); err != nil {
		return h.writeError(c, err)
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.accessPath(), i18n.T("users.custom_permissions_saved"))
}

func (h *Handler) deleteOverride(c fiber.Ctx) error {
	var request DeleteOverrideRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": i18n.T("common.invalid_payload")})
	}
	if err := h.service.DeleteOverride(
		c.Context(),
		c.Params("publicID"),
		request.Permission,
	); err != nil {
		return h.writeError(c, err)
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.accessPath(), i18n.T("users.custom_permissions_deleted"))
}
