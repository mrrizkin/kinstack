package auth

import (
	"errors"
	"strings"

	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/i18n"
	platformsession "kinstack/internal/platform/session"
	"kinstack/internal/platform/sessionregistry"

	"github.com/gofiber/fiber/v3"
)

// login validates username/password credentials, applies the second-factor
// challenge when required, and establishes the browser session.
func (h *Handler) login(c fiber.Ctx) error {
	var request LoginRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": i18n.T("common.invalid_payload"),
		})
	}

	session, err := h.service.LoginFrom(
		c.Context(),
		request.Username,
		request.Password,
		c.IP(),
	)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			props := h.loginProps()
			fieldErrors := fiber.Map{}
			if strings.TrimSpace(request.Username) == "" {
				fieldErrors["username"] = i18n.T("auth.username_required")
			}
			if request.Password == "" {
				fieldErrors["password"] = i18n.T("auth.password_required")
			}
			props["errors"] = fieldErrors
			return c.Status(fiber.StatusUnprocessableEntity).Render("auth/LoginView", props)
		case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrInactiveUser):
			// A disabled account must look exactly like a wrong password: the
			// password was verified, so the only difference is account state.
			if errors.Is(err, ErrInactiveUser) {
				h.logger.Infow(
					"sign-in rejected for inactive account",
					"request_id", c.RequestID(),
					"user", strings.TrimSpace(request.Username),
				)
			}
			props := h.loginProps()
			props["errors"] = fiber.Map{"auth": i18n.T("auth.invalid_credentials")}
			return c.Status(fiber.StatusUnprocessableEntity).Render("auth/LoginView", props)
		case errors.Is(err, ErrAccountLocked):
			props := h.loginProps()
			props["errors"] = fiber.Map{"auth": err.Error()}
			return c.Status(fiber.StatusTooManyRequests).Render("auth/LoginView", props)
		default:
			h.logger.Errorw("login failed", "request_id", c.RequestID(), "error", err)
			return fiber.ErrInternalServerError
		}
	}

	return h.completeLogin(c, session)
}

// twoFactorPage renders the second-factor challenge for a valid password that
// still needs a TOTP or recovery code.
func (h *Handler) twoFactorPage(c fiber.Ctx) error {
	if _, ok := platformsession.PendingTwoFactorActor(c); !ok {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	if h.actor.ActorFromContext(c) != nil {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.homePath(false))
	}
	err := c.Render("auth/TwoFactorView", fiber.Map{})
	if err != nil {
		h.logger.Errorw("render two-factor page", "request_id", c.RequestID(), "error", err)
	}
	return err
}

// verifyTwoFactor completes a pending challenge and establishes the session.
func (h *Handler) verifyTwoFactor(c fiber.Ctx) error {
	publicID, ok := platformsession.PendingTwoFactorActor(c)
	if !ok {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	if h.twoFactor == nil {
		return fiber.ErrServiceUnavailable
	}
	var request TwoFactorRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": i18n.T("auth.two_factor_invalid"),
		})
	}
	verified, err := h.twoFactor.Verify(c.Context(), publicID, request.Code)
	if err != nil {
		h.logger.Errorw("verify two-factor code", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	if !verified {
		return c.Status(fiber.StatusUnprocessableEntity).Render("auth/TwoFactorView", fiber.Map{
			"errors": fiber.Map{"code": i18n.T("auth.invalid_code")},
		})
	}
	session, err := h.service.Session(c.Context(), publicID)
	if err != nil {
		h.logger.Errorw("load two-factor session", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	platformsession.ClearTwoFactorChallenge(c)
	return h.establishSession(c, session)
}

// completeLogin applies the shared second-factor challenge and then establishes
// the authenticated session.
func (h *Handler) completeLogin(c fiber.Ctx, session SessionResponse) error {
	if !session.Superuser && h.twoFactor != nil {
		enabled, err := h.twoFactor.IsEnabled(c.Context(), session.PublicID)
		if err != nil {
			h.logger.Errorw("check two-factor status", "request_id", c.RequestID(), "error", err)
			return fiber.ErrInternalServerError
		}
		if enabled {
			if err := platformsession.SetTwoFactorChallenge(c, session.PublicID); err != nil {
				h.logger.Errorw(
					"establish two-factor challenge",
					"request_id",
					c.RequestID(),
					"error",
					err,
				)
				return fiber.ErrInternalServerError
			}
			return c.Redirect().Status(fiber.StatusSeeOther).To(h.twoFactorPath())
		}
	}
	return h.establishSession(c, session)
}

// establishSession stores the authenticated actor, tracks the browser session,
// and redirects to the signed-in landing page.
func (h *Handler) establishSession(c fiber.Ctx, session SessionResponse) error {
	if err := h.setSession(c, session); err != nil {
		return err
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(h.homePath(session.Superuser))
}

// setSession stores the authenticated actor and tracks the browser session
// without redirecting. Non-password authenticators use it to finish sign-in.
func (h *Handler) setSession(c fiber.Ctx, session SessionResponse) error {
	if err := h.actor.SetActor(c, actor.SessionActor{
		PublicID:    session.PublicID,
		Username:    session.Username,
		DisplayName: session.DisplayName,
		Role:        session.Role,
		Superuser:   session.Superuser,
	}); err != nil {
		h.logger.Errorw("establish session", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	h.registerSession(c, session)
	return nil
}

// cancelTwoFactor abandons a pending challenge and returns to the login page.
func (h *Handler) cancelTwoFactor(c fiber.Ctx) error {
	platformsession.ClearTwoFactorChallenge(c)
	return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
}

// registerSession records the freshly authenticated browser session so it can
// be listed and revoked from the profile security page. The ephemeral
// break-glass super admin has no account and therefore no tracked session.
func (h *Handler) registerSession(c fiber.Ctx, session SessionResponse) {
	if h.sessions == nil || session.Superuser {
		return
	}
	if _, err := h.sessions.Register(c.Context(), sessionregistry.RegisterInput{
		UserPublicID: session.PublicID,
		SessionID:    platformsession.CurrentID(c),
		IP:           c.IP(),
		UserAgent:    c.Get(fiber.HeaderUserAgent),
	}); err != nil {
		h.logger.Errorw("register browser session", "request_id", c.RequestID(), "error", err)
	}
}

// logout clears the actor and revokes the tracked browser session.
func (h *Handler) logout(c fiber.Ctx) error {
	sessionID := platformsession.CurrentID(c)
	if err := h.actor.ClearActor(c); err != nil {
		h.logger.Errorw("clear session", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
	if h.sessions != nil {
		if err := h.sessions.RevokeSession(c.Context(), sessionID); err != nil {
			h.logger.Errorw("revoke browser session", "request_id", c.RequestID(), "error", err)
		}
	}
	// Root resolves to the auth path for signed-out visitors.
	return c.Redirect().Status(fiber.StatusSeeOther).To("/")
}
