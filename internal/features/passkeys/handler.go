package passkeys

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/pkg/pagination"
	platformactor "kinstack/internal/platform/actor"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gofiber/fiber/v3"
	fibersession "github.com/gofiber/fiber/v3/middleware/session"
)

const passkeysPath = "/passkeys"

// ceremonySessionKey names the session entry holding one pending WebAuthn ceremony.
const ceremonySessionKey = "auth.passkey.ceremony"

// SessionStarter signs an account in after a verified credential assertion. The
// composition root wires the auth slice behind it, so this slice never imports
// another feature.
type SessionStarter interface {
	StartSession(context.Context, fiber.Ctx, string) error
}

type HandlerDependency struct {
	Service  *Service
	WebAuthn *webauthn.WebAuthn
	Sessions SessionStarter
	Actor    *platformactor.Actor
	PageSize pagination.PageSizeReader
	Logger   applog.Log
}

type Handler struct {
	service  *Service
	webAuthn *webauthn.WebAuthn
	sessions SessionStarter
	actor    *platformactor.Actor
	pageSize pagination.PageSizeReader
	logger   applog.Log
}

func NewHandler(d HandlerDependency) *Handler {
	assert.Assert(d.Service != nil, "passkeys service is required")
	assert.Assert(d.WebAuthn != nil, "passkeys WebAuthn configuration is required")
	assert.Assert(d.Sessions != nil, "passkeys session starter is required")
	assert.Assert(d.Actor != nil, "passkeys actor boundary is required")
	assert.Assert(d.Logger != nil, "passkeys logger is required")

	return &Handler{
		service:  d.Service,
		webAuthn: d.WebAuthn,
		sessions: d.Sessions,
		actor:    d.Actor,
		pageSize: d.PageSize,
		logger:   d.Logger,
	}
}

// MountPublic registers the unauthenticated passkey sign-in routes.
func (h *Handler) MountPublic(router fiber.Router) {
	router.Post("/login/passkey/options", h.beginPasskeyLogin)
	router.Post("/login/passkey", h.finishPasskeyLogin)
}

// MountWorkspace registers the authenticated credential-management routes.
func (h *Handler) MountWorkspace(router fiber.Router) {
	router.Get("/passkeys", h.passkeys)
	router.Post("/passkeys/register/options", h.beginPasskeyRegistration)
	router.Post("/passkeys/register", h.finishPasskeyRegistration)
	router.Post("/passkeys/:credentialID/delete", h.deletePasskey)
}

func (h *Handler) internalServerError(c fiber.Ctx, message string, err error) error {
	h.logger.Errorw(message, "request_id", c.RequestID(), "error", err)
	return fiber.NewError(fiber.StatusInternalServerError, message)
}

func (h *Handler) passkeys(c fiber.Ctx) error {
	actor := h.actor.ActorFromContext(c)
	if actor == nil {
		return fiber.ErrUnauthorized
	}
	if actor.Superuser {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/dashboard")
	}
	items, err := h.service.ListPasskeys(c.Context(), actor.PublicID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return fiber.ErrUnauthorized
		}
		return h.internalServerError(c, "unable to load passkeys", err)
	}
	// Passkeys per account are few, so the window is applied to the loaded slice
	// rather than pushed into the credential query.
	query, err := pagination.ParseRequest(
		c.Query("page"),
		c.Query("per_page"),
		pagination.ResolvePageSize(c.Context(), h.pageSize, actor.PublicID),
	)
	if err != nil {
		return c.Redirect().Status(fiber.StatusSeeOther).To(passkeysPath)
	}
	query = pagination.ClampPage(query, len(items))
	start, end := pagination.Window(query, len(items))
	return c.Render("passkeys/IndexView", fiber.Map{
		"passkeys": items[start:end],
		"page":     pagination.NewPage(query, len(items)),
	})
}

type passkeyCeremony struct {
	Kind      string               `json:"kind"`
	ActorID   string               `json:"actor_id,omitempty"`
	Session   webauthn.SessionData `json:"session"`
	CreatedAt time.Time            `json:"created_at"`
}

func savePasskeyCeremony(c fiber.Ctx, ceremony passkeyCeremony) error {
	store := fibersession.FromContext(c)
	if store == nil {
		return ErrPasskeyUnavailable
	}
	encoded, err := json.Marshal(ceremony)
	if err != nil {
		return err
	}
	store.Set(ceremonySessionKey, string(encoded))
	return nil
}

func consumePasskeyCeremony(c fiber.Ctx, kind, actor string) (passkeyCeremony, error) {
	store := fibersession.FromContext(c)
	if store == nil {
		return passkeyCeremony{}, ErrPasskeyUnavailable
	}
	value, _ := store.Get(ceremonySessionKey).(string)
	store.Delete(ceremonySessionKey)
	var ceremony passkeyCeremony
	if value == "" || json.Unmarshal([]byte(value), &ceremony) != nil || ceremony.Kind != kind ||
		!ceremony.CreatedAt.Add(10*time.Minute).After(time.Now().UTC()) ||
		(actor != "" && ceremony.ActorID != actor) {
		return passkeyCeremony{}, ErrPasskeyInvalid
	}
	return ceremony, nil
}

func fiberRequest(c fiber.Ctx) (*http.Request, error) {
	host := c.Hostname()
	if host == "" {
		host = "localhost"
	}
	request, err := http.NewRequestWithContext(
		c.Context(), c.Method(), "http://"+host+c.OriginalURL(), bytes.NewReader(c.Body()),
	)
	if err != nil {
		return nil, err
	}
	for key, value := range c.Request().Header.All() {
		request.Header.Add(string(key), string(value))
	}
	request.Host = host
	return request, nil
}

func (h *Handler) beginPasskeyRegistration(c fiber.Ctx) error {
	actor := h.actor.ActorFromContext(c)
	if actor == nil {
		return fiber.ErrUnauthorized
	}
	// Enrollment is credential management: a personal API token must never be
	// able to register a passkey, which would outlive the token itself.
	if actor.Scopes != nil {
		return fiber.ErrForbidden
	}
	if actor.Superuser {
		return fiber.ErrForbidden
	}
	user, err := h.service.FindByPublicID(c.Context(), actor.PublicID)
	if err != nil {
		return fiber.ErrUnauthorized
	}
	records, err := h.service.PasskeyRecords(c.Context(), user.PublicID)
	if err != nil {
		return h.internalServerError(c, "unable to load passkeys", err)
	}
	credentials := make([]webauthn.Credential, 0, len(records))
	for _, record := range records {
		credentials = append(credentials, record.Credential)
	}
	options, ceremony, err := passkeyRegistrationOptions(
		h.webAuthn,
		&passkeyUser{user: user, credentials: credentials},
		credentials,
	)
	if err != nil {
		return h.internalServerError(c, "unable to create passkey challenge", err)
	}
	if err := savePasskeyCeremony(c, passkeyCeremony{
		Kind: "register", ActorID: actor.PublicID, Session: *ceremony, CreatedAt: time.Now().UTC(),
	}); err != nil {
		return h.internalServerError(c, "unable to save passkey challenge", err)
	}
	return c.JSON(options)
}

func (h *Handler) finishPasskeyRegistration(c fiber.Ctx) error {
	actor := h.actor.ActorFromContext(c)
	if actor == nil {
		return fiber.ErrUnauthorized
	}
	if actor.Scopes != nil {
		return fiber.ErrForbidden
	}
	if actor.Superuser {
		return fiber.ErrForbidden
	}
	ceremony, err := consumePasskeyCeremony(c, "register", actor.PublicID)
	if err != nil {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			"passkey registration has expired; start again",
		)
	}
	var input PasskeyRegistrationRequest
	if err := json.Unmarshal(c.Body(), &input); err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid passkey payload")
	}
	input.Name = strings.TrimSpace(input.Name)
	if len(input.Name) < 1 || len(input.Name) > maxPasskeyNameLength {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			"passkey name must be between 1 and 80 characters",
		)
	}
	user, err := h.service.FindByPublicID(c.Context(), actor.PublicID)
	if err != nil {
		return fiber.ErrUnauthorized
	}
	request, err := fiberRequest(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid passkey payload")
	}
	credential, err := h.webAuthn.FinishRegistration(
		&passkeyUser{user: user},
		ceremony.Session,
		request,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "passkey could not be verified")
	}
	if !credential.Flags.UserVerified {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			"passkey must verify the user (biometric or PIN) to be used for sign-in",
		)
	}
	record := PasskeyRecord{
		ID:         append([]byte(nil), credential.ID...),
		Name:       input.Name,
		Credential: *credential,
		CreatedAt:  time.Now().UTC(),
	}
	if err := h.service.SavePasskey(c.Context(), user.PublicID, record); err != nil {
		if errors.Is(err, ErrPasskeyConflict) {
			return fiber.NewError(fiber.StatusConflict, "passkey is already registered")
		}
		return h.internalServerError(c, "unable to save passkey", err)
	}
	return c.JSON(passkeySummary(record))
}

// passkeyRegistrationOptions builds the creation options. User verification is
// required because a passkey stands in for both authentication factors.
func passkeyRegistrationOptions(
	webAuthn *webauthn.WebAuthn,
	user webauthn.User,
	credentials []webauthn.Credential,
) (*protocol.CredentialCreation, *webauthn.SessionData, error) {
	return webAuthn.BeginMediatedRegistration(
		user,
		protocol.MediationDefault,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementPreferred,
			UserVerification: protocol.VerificationRequired,
		}),
		webauthn.WithExclusions(webauthn.Credentials(credentials).CredentialDescriptors()),
	)
}

// passkeyLoginOptions builds the assertion options with the same user
// verification requirement enforced again on the assertion flags.
func passkeyLoginOptions(
	webAuthn *webauthn.WebAuthn,
) (*protocol.CredentialAssertion, *webauthn.SessionData, error) {
	return webAuthn.BeginDiscoverableMediatedLogin(
		protocol.MediationDefault,
		webauthn.WithUserVerification(protocol.VerificationRequired),
	)
}

func (h *Handler) beginPasskeyLogin(c fiber.Ctx) error {
	options, ceremony, err := passkeyLoginOptions(h.webAuthn)
	if err != nil {
		return h.internalServerError(c, "unable to create passkey challenge", err)
	}
	if err := savePasskeyCeremony(c, passkeyCeremony{
		Kind: "login", Session: *ceremony, CreatedAt: time.Now().UTC(),
	}); err != nil {
		return h.internalServerError(c, "unable to save passkey challenge", err)
	}
	return c.JSON(options)
}

func (h *Handler) finishPasskeyLogin(c fiber.Ctx) error {
	ceremony, err := consumePasskeyCeremony(c, "login", "")
	if err != nil {
		return fiber.NewError(
			fiber.StatusUnprocessableEntity,
			"passkey sign-in has expired; start again",
		)
	}
	request, err := fiberRequest(c)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid passkey payload")
	}
	user, credential, err := h.webAuthn.FinishPasskeyLogin(
		func(rawID, userHandle []byte) (webauthn.User, error) {
			resolved, records, lookupErr := h.service.FindPasskeyUser(
				c.Context(),
				rawID,
				userHandle,
			)
			if lookupErr != nil {
				return nil, lookupErr
			}
			credentials := make([]webauthn.Credential, 0, len(records))
			for _, record := range records {
				credentials = append(credentials, record.Credential)
			}
			return &passkeyUser{user: resolved, credentials: credentials}, nil
		},
		ceremony.Session,
		request,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "passkey sign-in was not verified")
	}
	if !credential.Flags.UserVerified {
		return fiber.NewError(
			fiber.StatusUnauthorized,
			"passkey sign-in requires user verification",
		)
	}
	resolved := user.(*passkeyUser)
	updated := PasskeyRecord{
		ID: credential.ID, Credential: *credential, LastUsedAt: timePtr(time.Now().UTC()),
	}
	records, err := h.service.PasskeyRecords(c.Context(), resolved.user.PublicID)
	if err != nil {
		return h.internalServerError(c, "unable to update passkey", err)
	}
	for _, record := range records {
		if bytes.Equal(record.ID, credential.ID) {
			updated.Name = record.Name
			updated.CreatedAt = record.CreatedAt
			break
		}
	}
	if err := h.service.UpdatePasskey(c.Context(), resolved.user.PublicID, updated); err != nil {
		return h.internalServerError(c, "unable to update passkey", err)
	}
	if err := h.sessions.StartSession(c.Context(), c, resolved.user.PublicID); err != nil {
		return h.internalServerError(c, "unable to create session", err)
	}
	// Passkeys satisfy both authentication factors per the WebAuthn spec
	// (possession of the device + biometric/PIN unlock). A separate 2FA
	// challenge is therefore not required after a successful passkey assertion.
	return c.JSON(fiber.Map{"ok": true})
}

func (h *Handler) deletePasskey(c fiber.Ctx) error {
	actor := h.actor.ActorFromContext(c)
	if actor == nil {
		return fiber.ErrUnauthorized
	}
	if actor.Scopes != nil {
		return fiber.ErrForbidden
	}
	if actor.Superuser {
		return fiber.ErrForbidden
	}
	credentialID, err := base64.RawURLEncoding.DecodeString(c.Params("credentialID"))
	if err != nil || len(credentialID) == 0 {
		return fiber.NewError(fiber.StatusUnprocessableEntity, "invalid passkey identifier")
	}
	if err := h.service.DeletePasskey(c.Context(), actor.PublicID, credentialID); err != nil {
		if errors.Is(err, ErrPasskeyNotFound) {
			return fiber.ErrNotFound
		}
		return h.internalServerError(c, "unable to revoke passkey", err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func timePtr(value time.Time) *time.Time { return &value }
