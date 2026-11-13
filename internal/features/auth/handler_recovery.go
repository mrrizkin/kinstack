package auth

import (
	"errors"

	"kinstack/internal/pkg/log"

	"kinstack/internal/platform/i18n"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) forgotPasswordPage(c fiber.Ctx) error {
	if !h.passwordReset {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	err := c.Render("auth/ForgotPasswordView", fiber.Map{})
	if err != nil {
		log.ForRequest(h.logger, c.RequestID()).Errorw("render forgot password page", "error", err)
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
		log.ForRequest(h.logger, c.RequestID()).Errorw("request password reset", "error", err)
		return fiber.ErrInternalServerError
	}
	return c.Render("auth/ForgotPasswordView", fiber.Map{
		"notice": i18n.T("auth.reset_link_sent"),
	})
}

func (h *Handler) resetPasswordPage(c fiber.Ctx) error {
	err := c.Render("auth/ResetPasswordView", fiber.Map{"token": c.Params("token")})
	if err != nil {
		log.ForRequest(h.logger, c.RequestID()).Errorw("render reset password page", "error", err)
	}
	return err
}

func (h *Handler) resetPassword(c fiber.Ctx) error {
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
			return c.Status(fiber.StatusUnprocessableEntity).
				Render("auth/ResetPasswordView", fiber.Map{
					"token":  token,
					"errors": fiber.Map{"new_password": err.Error()},
				})
		default:
			log.ForRequest(h.logger, c.RequestID()).Errorw("reset password", "error", err)
			return fiber.ErrInternalServerError
		}
	}
	return c.Render("auth/ResetPasswordView", fiber.Map{"done": true})
}
