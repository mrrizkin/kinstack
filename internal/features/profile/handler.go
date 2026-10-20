package profile

import (
	"errors"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/i18n"
	platforminertia "kinstack/internal/platform/inertia"
	platformsession "kinstack/internal/platform/session"
	"kinstack/internal/platform/sessionregistry"
	"kinstack/internal/platform/twofactor"

	"codeberg.org/mrrizkin/fibertia"
	"github.com/gofiber/fiber/v3"
)

// HandlerDependency contains the service, actor boundary, session registry,
// Inertia adapter, and logger used by the handler.
type HandlerDependency struct {
	Service   *Service
	Actor     *actor.Actor
	Logger    log.Log
	Inertia   *fibertia.Fibertia
	Sessions  *sessionregistry.Service
	TwoFactor *twofactor.Service
}

// Handler owns the HTTP boundary for the profile page.
type Handler struct {
	service   *Service
	actor     *actor.Actor
	logger    log.Log
	inertia   *fibertia.Fibertia
	sessions  *sessionregistry.Service
	twoFactor *twofactor.Service
}

// NewHandler constructs the profile HTTP handler.
func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "profile service is required")
	assert.Assert(d.Actor != nil, "profile actor boundary is required")
	assert.Assert(d.Logger != nil, "profile logger is required")
	assert.Assert(d.Inertia != nil, "profile inertia adapter is required")

	return &Handler{
		service:   d.Service,
		actor:     d.Actor,
		logger:    d.Logger,
		inertia:   d.Inertia,
		sessions:  d.Sessions,
		twoFactor: d.TwoFactor,
	}
}

func (h *Handler) profilePath() string {
	return "/profile"
}

func (h *Handler) dashboardPath() string {
	return "/dashboard"
}

func (h *Handler) actorPublicID(c fiber.Ctx) string {
	if sessionActor := h.actor.ActorFromContext(c); sessionActor != nil {
		return sessionActor.PublicID
	}
	return ""
}

// Mount registers profile routes on the authenticated workspace group.
func (h *Handler) Mount(router fiber.Router) {
	router.Get("/profile", h.show)
	router.Post("/profile", h.updateProfile)
	router.Post("/profile/preferences", h.updatePreferences)
	router.Post("/profile/password", h.changePassword)
	router.Post("/profile/sessions/:publicID/revoke", h.revokeSession)
	router.Post("/profile/sessions/revoke-others", h.revokeOtherSessions)
	router.Post("/profile/two-factor/begin", h.beginTwoFactor)
	router.Post("/profile/two-factor/confirm", h.confirmTwoFactor)
	router.Post("/profile/two-factor/disable", h.disableTwoFactor)
	router.Post("/profile/two-factor/recovery-codes", h.regenerateRecoveryCodes)
}

func (h *Handler) show(c fiber.Ctx) error {
	// The ephemeral break-glass super admin has no database profile; send it to
	// the dashboard instead.
	if session := h.actor.ActorFromContext(c); session != nil && session.Superuser {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.dashboardPath())
	}
	props, err := h.service.Show(c.Context(), h.actorPublicID(c))
	if err != nil {
		return h.writeError(c, err, "form")
	}
	props.Session = SessionDTO{
		IP:        c.IP(),
		UserAgent: c.Get(fiber.HeaderUserAgent),
		LoginAt:   props.Profile.LastLoginAt,
	}
	return c.Render("profile/IndexView", fiber.Map{
		"profile":     props.Profile,
		"preferences": props.Preferences,
		"session":     props.Session,
		"sessions":    h.activeSessions(c),
		"two_factor":  h.twoFactorState(c, props),
	})
}

// twoFactorState builds the security-tab contract, including the one-time
// recovery codes that were just issued during enrollment or rotation.
func (h *Handler) twoFactorState(c fiber.Ctx, props ProfilePageProps) TwoFactorDTO {
	state := TwoFactorDTO{RecoveryCodes: []string{}}
	if h.twoFactor == nil {
		return state
	}
	status, err := h.twoFactor.Status(c.Context(), h.actorPublicID(c))
	if err != nil {
		h.logger.Errorw("load two-factor status", "request_id", c.RequestID(), "error", err)
		return state
	}
	state.Enabled = status.Enabled
	if !status.Enabled && strings.TrimSpace(status.Secret) != "" {
		state.PendingSecret = status.Secret
		state.URI = h.twoFactor.ProvisioningURI(status.Secret, props.Profile.Username)
	}
	if codes, ok := platformsession.ConsumeRecoveryCodes(c); ok {
		state.RecoveryCodes = codes
	}
	return state
}

// activeSessions lists the signed-in browser sessions for the security tab.
func (h *Handler) activeSessions(c fiber.Ctx) []ActiveSessionDTO {
	sessions := make([]ActiveSessionDTO, 0)
	if h.sessions == nil {
		return sessions
	}
	results, err := h.sessions.List(c.Context(), h.actorPublicID(c))
	if err != nil {
		h.logger.Errorw("list active sessions", "request_id", c.RequestID(), "error", err)
		return sessions
	}
	currentID := platformsession.CurrentID(c)
	for _, result := range results {
		sessions = append(sessions, ActiveSessionDTO{
			PublicID:   result.PublicID,
			IP:         result.IP,
			UserAgent:  result.UserAgent,
			LoginAt:    result.CreatedAt.UTC().Format(time.RFC3339),
			LastSeenAt: result.LastSeenAt.UTC().Format(time.RFC3339),
			Current:    result.SessionID == currentID,
		})
	}
	return sessions
}

func (h *Handler) updateProfile(c fiber.Ctx) error {
	var request UpdateProfileRequest
	if err := c.Bind().Body(&request); err != nil {
		return h.validationError(c, i18n.T("common.invalid_payload"), "form")
	}
	if _, err := h.service.UpdateDisplayName(c.Context(), h.actorPublicID(c), request); err != nil {
		return h.writeError(c, err, "display_name")
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.profilePath(), i18n.T("profile.updated"))
}

func (h *Handler) updatePreferences(c fiber.Ctx) error {
	var request UpdatePreferencesRequest
	if err := c.Bind().Body(&request); err != nil {
		return h.validationError(c, i18n.T("common.invalid_payload"), "form")
	}
	if _, err := h.service.UpdatePreferences(c.Context(), h.actorPublicID(c), request); err != nil {
		return h.writeError(c, err, "theme")
	}
	return platforminertia.RedirectWithFlash(h.inertia, c, h.profilePath(), i18n.T("profile.preferences_saved"))
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
	return platforminertia.RedirectWithFlash(h.inertia, c, h.profilePath(), i18n.T("profile.password_changed"))
}

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
	return platforminertia.RedirectWithFlash(h.inertia, c, h.profilePath(), i18n.T("profile.session_revoked"))
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
	return platforminertia.RedirectWithFlash(h.inertia, c, h.profilePath(), i18n.T("profile.other_sessions_revoked"))
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
	return platforminertia.RedirectWithFlash(
		h.inertia,
		c,
		h.profilePath(),
		i18n.T("profile.two_factor_scan"),
	)
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
	return platforminertia.RedirectWithFlash(
		h.inertia,
		c,
		h.profilePath(),
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
	return platforminertia.RedirectWithFlash(
		h.inertia,
		c,
		h.profilePath(),
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
	return platforminertia.RedirectWithFlash(
		h.inertia,
		c,
		h.profilePath(),
		i18n.T("profile.recovery_codes_created"),
	)
}

// writeTwoFactorError maps two-factor sentinel errors onto form fields.
func (h *Handler) writeTwoFactorError(c fiber.Ctx, err error, field string) error {
	switch {
	case errors.Is(err, twofactor.ErrInvalidCode):
		return h.validationError(c, i18n.T("profile.invalid_code"), "code")
	case errors.Is(err, twofactor.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
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

func (h *Handler) writeError(c fiber.Ctx, err error, field string) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return h.validationError(c, err.Error(), field)
	case errors.Is(err, ErrWrongPassword):
		return h.validationError(c, i18n.T("profile.current_password_wrong"), "current_password")
	case errors.Is(err, ErrPasswordMismatch):
		return h.validationError(c, i18n.T("profile.confirm_mismatch"), "confirm_password")
	case errors.Is(err, ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	default:
		h.logger.Errorw("profile request failed", "request_id", c.RequestID(), "error", err)
		return fiber.ErrInternalServerError
	}
}

// passwordField maps a validation error onto the password form field.
func passwordField(err error) string {
	if errors.Is(err, ErrWrongPassword) {
		return "current_password"
	}
	if errors.Is(err, ErrPasswordMismatch) {
		return "confirm_password"
	}
	return "new_password"
}

// validationError flashes the message both for the field and the page banner.
func (h *Handler) validationError(c fiber.Ctx, message string, field string) error {
	message = strings.TrimSpace(message)
	fields := map[string]string{"form": message}
	if field != "" && field != "form" {
		fields[field] = message
	}
	return platforminertia.ValidationErrors(h.inertia, c, h.profilePath(), fields)
}
