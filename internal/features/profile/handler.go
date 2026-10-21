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

	"github.com/gofiber/fiber/v3"
)

// HandlerDependency contains the service, actor boundary, session registry,
// Inertia adapter, and logger used by the handler.
type HandlerDependency struct {
	Service   *Service
	Actor     *actor.Actor
	Logger    log.Log
	Inertia   *platforminertia.Adapter
	Sessions  *sessionregistry.Service
	TwoFactor *twofactor.Service
}

// Handler owns the HTTP boundary for the profile page.
type Handler struct {
	service   *Service
	actor     *actor.Actor
	logger    log.Log
	inertia   *platforminertia.Adapter
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

// profilePathFor keeps the settings navigation on the section that submitted a
// form, so validation errors and flash messages do not jump back to the profile
// tab.
func (h *Handler) profilePathFor(c fiber.Ctx) string {
	switch path := c.Path(); {
	case strings.HasSuffix(path, "/preferences"):
		return "/profile/preferences"
	case strings.HasPrefix(path, "/profile/password"),
		strings.Contains(path, "/sessions/"),
		strings.Contains(path, "/two-factor/"):
		return "/profile/security"
	default:
		return "/profile"
	}
}

// profileSection maps a dedicated settings route to the panel the page renders.
func profileSection(path string) string {
	switch path {
	case "/profile/security":
		return "security"
	case "/profile/preferences":
		return "preferences"
	default:
		return "profile"
	}
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
	router.Get("/profile/security", h.show)
	router.Get("/profile/preferences", h.show)
	router.Post("/profile", h.updateProfile)
	router.Post("/profile/preferences", h.updatePreferences)
	router.Post("/profile/theme", h.updateTheme)
	router.Post("/profile/password", h.changePassword)
	router.Post("/profile/sessions/:publicID/revoke", h.revokeSession)
	router.Post("/profile/sessions/revoke-others", h.revokeOtherSessions)
	router.Post("/profile/two-factor/begin", h.beginTwoFactor)
	router.Post("/profile/two-factor/confirm", h.confirmTwoFactor)
	router.Post("/profile/two-factor/disable", h.disableTwoFactor)
	router.Post("/profile/two-factor/recovery-codes", h.regenerateRecoveryCodes)
}

func (h *Handler) show(c fiber.Ctx) error {
	// Keep old tab URLs working after moving the settings tabs onto dedicated
	// routes.
	if c.Path() == "/profile" {
		switch section := c.Query("section"); section {
		case "security", "preferences":
			return c.Redirect().Status(fiber.StatusSeeOther).To("/profile/" + section)
		}
	}
	// The ephemeral break-glass super admin has no database profile; send it to
	// the dashboard instead.
	if session := h.actor.ActorFromContext(c); session != nil && session.Superuser {
		return c.Redirect().Status(fiber.StatusSeeOther).To(h.dashboardPath())
	}
	props, err := h.service.Show(c.Context(), h.actorPublicID(c))
	if err != nil {
		return h.writeError(c, err, "form")
	}
	switch profileSection(c.Path()) {
	case "security":
		return c.Render("profile/SecurityView", fiber.Map{
			"profile":    props.Profile,
			"sessions":   h.activeSessions(c),
			"two_factor": h.twoFactorState(c, props),
		})
	case "preferences":
		return c.Render("profile/PreferencesView", fiber.Map{
			"preferences": props.Preferences,
		})
	default:
		return c.Render("profile/ProfileView", fiber.Map{
			"profile": props.Profile,
		})
	}
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

func (h *Handler) writeError(c fiber.Ctx, err error, field string) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return h.validationError(c, err.Error(), field)
	case errors.Is(err, ErrWrongPassword):
		return h.validationError(c, i18n.T("profile.current_password_wrong"), "current_password")
	case errors.Is(err, ErrPasswordMismatch):
		return h.validationError(c, i18n.T("profile.confirm_mismatch"), "confirm_password")
	case errors.Is(err, ErrNotFound):
		return h.inertia.FailedAction(c,
			h.profilePathFor(c),
			fiber.StatusNotFound,
			err.Error(),
		)
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
	return h.inertia.ValidationErrors(c, h.profilePathFor(c), fields)
}
