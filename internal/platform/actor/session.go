package actor

import (
	"context"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
	fibersession "github.com/gofiber/fiber/v3/middleware/session"
)

const (
	actorLocalKey       = "actor"
	actorSessionKey     = "actor_public_id"
	actorUsernameKey    = "actor_username"
	actorDisplayNameKey = "actor_display_name"
	actorRoleKey        = "actor_role"
	actorSuperuserKey   = "actor_superuser"
)

// SuperuserPublicID identifies the ephemeral break-glass super admin session.
// It is not a database account: like basic auth, the credentials come from the
// environment and nothing is persisted.
const SuperuserPublicID = "break-glass"

// SessionActor is the authenticated actor carried by the browser session.
type SessionActor struct {
	PublicID    string   `json:"public_id"`
	Username    string   `json:"username,omitempty"`
	DisplayName string   `json:"display_name,omitempty"`
	Role        string   `json:"role,omitempty"`
	Superuser   bool     `json:"superuser,omitempty"`
	// Scopes carries a personal API token's permission allowlist. It is nil for
	// normal browser sessions and non-nil (possibly empty) for bearer tokens.
	Scopes []string `json:"scopes,omitempty"`
}

// SetActor establishes the authenticated actor in the current browser session.
func (a *Actor) SetActor(c fiber.Ctx, value SessionActor) error {
	middleware := fibersession.FromContext(c)
	if middleware == nil {
		return errors.New("session middleware is not available")
	}
	if err := middleware.Regenerate(); err != nil {
		return fmt.Errorf("regenerate session: %w", err)
	}
	middleware.Set(actorSessionKey, value.PublicID)
	middleware.Set(actorUsernameKey, value.Username)
	middleware.Set(actorDisplayNameKey, value.DisplayName)
	middleware.Set(actorRoleKey, value.Role)
	middleware.Set(actorSuperuserKey, value.Superuser)
	c.Locals(actorLocalKey, &value)
	return nil
}

// SetRequestActor attaches an actor resolved by a non-session authenticator
// (for example an API bearer token) to the current request. It is not persisted
// to the browser session and is discarded when the request ends.
func (a *Actor) SetRequestActor(c fiber.Ctx, value SessionActor) {
	c.Locals(actorLocalKey, &value)
}

// ClearActor destroys the current authenticated browser session.
func (a *Actor) ClearActor(c fiber.Ctx) error {
	middleware := fibersession.FromContext(c)
	if middleware == nil {
		return errors.New("session middleware is not available")
	}
	return middleware.Destroy()
}

// ActorFromContext returns the actor established by the browser session.
func (a *Actor) ActorFromContext(c fiber.Ctx) *SessionActor {
	if value, ok := c.Locals(actorLocalKey).(*SessionActor); ok {
		return value
	}
	middleware := fibersession.FromContext(c)
	if middleware == nil {
		return nil
	}
	publicID, _ := middleware.Get(actorSessionKey).(string)
	if publicID == "" {
		return nil
	}
	username, _ := middleware.Get(actorUsernameKey).(string)
	displayName, _ := middleware.Get(actorDisplayNameKey).(string)
	role, _ := middleware.Get(actorRoleKey).(string)
	superuser, _ := middleware.Get(actorSuperuserKey).(bool)
	value := &SessionActor{
		PublicID:    publicID,
		Username:    username,
		DisplayName: displayName,
		Role:        role,
		Superuser:   superuser,
	}
	c.Locals(actorLocalKey, value)
	return value
}

// ValidateSession verifies that the session actor still resolves to an active
// account. Ephemeral break-glass superuser sessions have no account and are
// trusted as long as the session itself is intact.
func (a *Actor) ValidateSession(ctx context.Context, value *SessionActor) error {
	if value == nil {
		return ErrInvalidActor
	}
	if value.Superuser {
		return nil
	}
	return a.Validate(ctx, value.PublicID)
}

// RequireKnownActor protects routes and rejects sessions whose actor no longer
// resolves to an active account.
func (a *Actor) RequireKnownActor() fiber.Handler {
	return func(c fiber.Ctx) error {
		value := a.ActorFromContext(c)
		if value == nil {
			return a.RequireAuth(c)
		}
		if a.ValidateSession(c.Context(), value) != nil {
			_ = a.ClearActor(c)
			return fiber.ErrUnauthorized
		}
		return c.Next()
	}
}

// Current returns the session actor after validating it against the actor
// boundary. Invalid actors are removed from the browser session.
func (a *Actor) Current(c fiber.Ctx) (*SessionActor, error) {
	value := a.ActorFromContext(c)
	if value == nil {
		return nil, fiber.ErrUnauthorized
	}
	if err := a.ValidateSession(c.Context(), value); err != nil {
		_ = a.ClearActor(c)
		return nil, fiber.ErrUnauthorized
	}
	return value, nil
}

// RequireAuth protects routes that need an authenticated actor.
func (a *Actor) RequireAuth(c fiber.Ctx) error {
	if a.ActorFromContext(c) != nil {
		return c.Next()
	}
	if c.Method() == fiber.MethodGet || c.Get("X-Inertia") == "true" {
		return c.Redirect().Status(fiber.StatusSeeOther).To("/login")
	}
	return fiber.ErrUnauthorized
}
