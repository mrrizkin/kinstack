package auth

import (
	"errors"
	"strings"

	"kinstack/internal/pkg/log"

	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/i18n"
	platformsession "kinstack/internal/platform/session"
	"kinstack/internal/platform/sessionregistry"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) login(c fiber.Ctx) error {
	var request LoginRequest
	if err := c.Bind().Body(&request); err != nil {
		props := h.loginProps()
		props["errors"] = fiber.Map{"auth": i18n.T("common.invalid_payload")}
		return c.Status(fiber.StatusBadRequest).Render("auth/LoginView", props)
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
				log.ForRequest(h.logger, c.RequestID()).Infow(
					"sign-in rejected for inactive account",
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
			log.ForRequest(h.logger, c.RequestID()).Errorw("login failed", "error", err)
			return fiber.ErrInternalServerError
		}
	}

	return h.completeLogin(c, session)
}

func (h *Handler) twoFactorPage(c fiber.Ctx) error {
	if _, ok := platformsession.PendingTwoFactorActor(c); !ok {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	if h.actor.ActorFromContext(c) != nil {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.homePath(false))
	}
	err := c.Render("auth/TwoFactorView", fiber.Map{})
	if err != nil {
		log.ForRequest(h.logger, c.RequestID()).Errorw("render two-factor page", "error", err)
	}
	return err
}

func (h *Handler) verifyTwoFactor(c fiber.Ctx) error {
	publicID, ok := platformsession.PendingTwoFactorActor(c)
	if !ok {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
	}
	if h.twoFactor == nil {
		return fiber.ErrServiceUnavailable
	}
	client := c.IP()
	if !h.service.SecondFactorAllowed(c.Context(), publicID, client) {
		return c.Status(fiber.StatusTooManyRequests).Render("auth/TwoFactorView", fiber.Map{
			"errors": fiber.Map{"code": i18n.T("auth.account_locked")},
		})
	}
	var request TwoFactorRequest
	if err := c.Bind().Body(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).Render("auth/TwoFactorView", fiber.Map{
			"errors": fiber.Map{"code": i18n.T("auth.two_factor_invalid")},
		})
	}
	verified, err := h.twoFactor.Verify(c.Context(), publicID, request.Code)
	if err != nil {
		log.ForRequest(h.logger, c.RequestID()).Errorw("verify two-factor code", "error", err)
		return fiber.ErrInternalServerError
	}
	if !verified {
		h.service.RecordSecondFactorFailure(c.Context(), publicID, client)
		return c.Status(fiber.StatusUnprocessableEntity).Render("auth/TwoFactorView", fiber.Map{
			"errors": fiber.Map{"code": i18n.T("auth.invalid_code")},
		})
	}
	h.service.ClearSecondFactorFailures(c.Context(), publicID, client)
	session, err := h.service.Session(c.Context(), publicID)
	if err != nil {
		log.ForRequest(h.logger, c.RequestID()).Errorw("load two-factor session", "error", err)
		return fiber.ErrInternalServerError
	}
	platformsession.ClearTwoFactorChallenge(c)
	return h.establishSession(c, session)
}

func (h *Handler) completeLogin(c fiber.Ctx, session SessionResponse) error {
	if !session.Superuser && h.twoFactor != nil {
		enabled, err := h.twoFactor.IsEnabled(c.Context(), session.PublicID)
		if err != nil {
			log.ForRequest(h.logger, c.RequestID()).Errorw("check two-factor status", "error", err)
			return fiber.ErrInternalServerError
		}
		if enabled {
			if err := platformsession.SetTwoFactorChallenge(c, session.PublicID); err != nil {
				log.ForRequest(h.logger, c.RequestID()).Errorw(
					"establish two-factor challenge",
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

func (h *Handler) establishSession(c fiber.Ctx, session SessionResponse) error {
	if err := h.setSession(c, session); err != nil {
		return err
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To(h.homePath(session.Superuser))
}

func (h *Handler) setSession(c fiber.Ctx, session SessionResponse) error {
	if err := h.actor.SetActor(c, actor.SessionActor{
		PublicID:    session.PublicID,
		Username:    session.Username,
		DisplayName: session.DisplayName,
		Role:        session.Role,
		Superuser:   session.Superuser,
	}); err != nil {
		log.ForRequest(h.logger, c.RequestID()).Errorw("establish session", "error", err)
		return fiber.ErrInternalServerError
	}
	h.registerSession(c, session)
	return nil
}

func (h *Handler) cancelTwoFactor(c fiber.Ctx) error {
	platformsession.ClearTwoFactorChallenge(c)
	return c.Redirect().Status(fiber.StatusSeeOther).To(h.loginPath())
}

// registerSession records the session so it can be listed and revoked; the break-glass
// super admin has no account and therefore no tracked session.
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
		log.ForRequest(h.logger, c.RequestID()).Errorw("register browser session", "error", err)
	}
}

func (h *Handler) logout(c fiber.Ctx) error {
	sessionID := platformsession.CurrentID(c)
	if err := h.actor.ClearActor(c); err != nil {
		log.ForRequest(h.logger, c.RequestID()).Errorw("clear session", "error", err)
		return fiber.ErrInternalServerError
	}
	if h.sessions != nil {
		if err := h.sessions.RevokeSession(c.Context(), sessionID); err != nil {
			log.ForRequest(h.logger, c.RequestID()).Errorw("revoke browser session", "error", err)
		}
	}
	return c.Redirect().Status(fiber.StatusSeeOther).To("/")
}
