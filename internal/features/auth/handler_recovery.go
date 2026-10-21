package auth

import (
	"errors"

	"kinstack/internal/platform/i18n"

	"github.com/gofiber/fiber/v3"
)

// requestMagicLink emails a single-use sign-in link. The response is identical
// whether or not the address belongs to an account.
func (h *Handler) requestMagicLink(c fiber.Ctx) error {
	var request MagicLinkRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": i18n.T("common.invalid_payload"),
		})
	}
	if err := h.service.RequestMagicLink(c.Context(), request.Email); err != nil {
		if errors.Is(err, ErrInvalidInput) {
			return c.Status(fiber.StatusUnprocessableEntity).Render("auth/LoginView", fiber.Map{
				"errors": fiber.Map{"email": i18n.T("auth.email_invalid")},
			})
		}
		h.logger.Errorw("request magic link", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return c.Render("auth/LoginView", fiber.Map{
		"notice": i18n.T("auth.magic_link_sent"),
	})
}

// consumeMagicLink verifies a sign-in token and continues the normal sign-in
// flow (including the second factor when enabled).
func (h *Handler) consumeMagicLink(c fiber.Ctx) error {
	session, err := h.service.ConsumeMagicLink(c.Context(), c.Params("token"))
	if err != nil {
		if errors.Is(err, ErrInvalidToken) || errors.Is(err, ErrInvalidInput) {
			return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
		}
		h.logger.Errorw("consume magic link", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return h.completeLogin(c, session)
}

// forgotPasswordPage renders the self-service reset request form.
func (h *Handler) forgotPasswordPage(c fiber.Ctx) error {
	if !h.passwordReset {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	err := c.Render("auth/ForgotPasswordView", fiber.Map{})
	if err != nil {
		h.logger.Errorw("render forgot password page", "request_id", c.RequestID(), "error", err)
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
		h.logger.Errorw("request password reset", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	return c.Render("auth/ForgotPasswordView", fiber.Map{
		"notice": i18n.T("auth.reset_link_sent"),
	})
}

// resetPasswordPage renders the new-password form for one reset token.
func (h *Handler) resetPasswordPage(c fiber.Ctx) error {
	if !h.passwordReset {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	err := c.Render("auth/ResetPasswordView", fiber.Map{"token": c.Params("token")})
	if err != nil {
		h.logger.Errorw("render reset password page", "request_id", c.RequestID(), "error", err)
	}
	return err
}

// resetPassword burns a reset token and stores the new password.
func (h *Handler) resetPassword(c fiber.Ctx) error {
	if !h.passwordReset {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
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
			h.logger.Errorw("reset password", "request_id", c.RequestID(), "error", err)
			return fiber.ErrInternalServerError
		}
	}
	return c.Render("auth/ResetPasswordView", fiber.Map{"done": true})
}
