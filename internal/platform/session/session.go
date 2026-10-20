// Package session configures Fiber session transport and request-scoped state.
package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"kinstack/config"
	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/log"

	"github.com/gofiber/fiber/v3"
	fibersession "github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/storage/valkey"
)

const (
	twoFactorSessionKey     = "two_factor_challenge"
	recoveryCodesSessionKey = "recovery_codes"
	twoFactorChallengeTTL   = 10 * time.Minute
)

type twoFactorChallenge struct {
	PublicID string    `json:"public_id"`
	IssuedAt time.Time `json:"issued_at"`
}

type StoreDependency struct {
	Config       config.ValkeyConfig
	Storage      fiber.Storage
	IdleTimeout  time.Duration
	CookieSecure bool
	Log          *log.Logger
}

// NewValkeyStorage creates the shared Valkey boundary used by sessions and
// distributed request protection.
func NewValkeyStorage(cfg config.ValkeyConfig) fiber.Storage {
	assert.Assert(cfg.Host != "", "valkey host is required")
	assert.Assert(cfg.Port > 0, "valkey port is required")

	return valkey.New(valkey.Config{
		InitAddress: []string{fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)},
		Password:    cfg.Password,
		SelectDB:    cfg.Database,
	})
}

// NewStore creates the production session store backed by Valkey.
func NewStore(d StoreDependency) *fibersession.Store {
	assert.Assert(d.Config.Host != "", "valkey host is required")
	assert.Assert(d.Config.Port > 0, "valkey port is required")
	assert.Assert(d.Log != nil, "logger is required")

	storage := d.Storage
	if storage == nil {
		storage = NewValkeyStorage(d.Config)
	}
	store := fibersession.NewStore(fibersession.Config{
		Storage:        storage,
		IdleTimeout:    d.IdleTimeout,
		CookiePath:     "/",
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
		CookieSecure:   d.CookieSecure,
	})
	d.Log.Infow("Session configured with Valkey storage")
	return store
}

// NewMemoryStore is used by HTTP tests and local wiring that does not need Valkey.
func NewMemoryStore(idleTimeout time.Duration) *fibersession.Store {
	return NewMemoryStoreWithCookieSecurity(idleTimeout, false)
}

// NewMemoryStoreWithCookieSecurity mirrors the production cookie settings for
// tests without requiring a Valkey process.
func NewMemoryStoreWithCookieSecurity(
	idleTimeout time.Duration,
	cookieSecure bool,
) *fibersession.Store {
	return fibersession.NewStore(fibersession.Config{
		IdleTimeout:    idleTimeout,
		CookiePath:     "/",
		CookieHTTPOnly: true,
		CookieSameSite: "Lax",
		CookieSecure:   cookieSecure,
	})
}

// NewMiddleware loads and saves one Fiber session for each request.
func NewMiddleware(store *fibersession.Store, idleTimeout time.Duration) fiber.Handler {
	assert.Assert(store != nil, "session store is required")

	return mustSessionMiddleware(store, idleTimeout)
}

func mustSessionMiddleware(store *fibersession.Store, idleTimeout time.Duration) fiber.Handler {
	handler, _ := fibersession.NewWithStore(fibersession.Config{
		Store:       store,
		IdleTimeout: idleTimeout,
	})
	return handler
}

// SetTwoFactorChallenge replaces the current session with a short-lived,
// pre-authentication challenge. No authenticated actor is exposed until the
// challenge is completed successfully.
func SetTwoFactorChallenge(c fiber.Ctx, publicID string) error {
	publicID = strings.TrimSpace(publicID)
	middleware := fibersession.FromContext(c)
	if middleware == nil || publicID == "" {
		return errors.New("session middleware is not available")
	}
	if err := middleware.Regenerate(); err != nil {
		return fmt.Errorf("regenerate two-factor session: %w", err)
	}
	encoded, err := json.Marshal(twoFactorChallenge{PublicID: publicID, IssuedAt: time.Now().UTC()})
	if err != nil {
		return fmt.Errorf("encode two-factor challenge: %w", err)
	}
	middleware.Set(twoFactorSessionKey, string(encoded))
	return nil
}

// PendingTwoFactorActor returns the actor bound to a live pre-authentication
// challenge without consuming it. Expired challenges are removed immediately.
func PendingTwoFactorActor(c fiber.Ctx) (string, bool) {
	middleware := fibersession.FromContext(c)
	if middleware == nil {
		return "", false
	}
	raw, _ := middleware.Get(twoFactorSessionKey).(string)
	var challenge twoFactorChallenge
	if raw == "" || json.Unmarshal([]byte(raw), &challenge) != nil ||
		strings.TrimSpace(challenge.PublicID) == "" ||
		challenge.IssuedAt.Before(time.Now().UTC().Add(-twoFactorChallengeTTL)) {
		if raw != "" {
			middleware.Delete(twoFactorSessionKey)
		}
		return "", false
	}
	return challenge.PublicID, true
}

// ClearTwoFactorChallenge removes a pre-authentication challenge after a
// successful verification or an explicit cancellation.
func ClearTwoFactorChallenge(c fiber.Ctx) {
	if middleware := fibersession.FromContext(c); middleware != nil {
		middleware.Delete(twoFactorSessionKey)
	}
}

// SetRecoveryCodes stores one-time plaintext recovery codes in the server-side
// session. They are only available immediately after setup or rotation.
func SetRecoveryCodes(c fiber.Ctx, codes []string) error {
	middleware := fibersession.FromContext(c)
	if middleware == nil {
		return errors.New("session middleware is not available")
	}
	encoded, err := json.Marshal(codes)
	if err != nil {
		return fmt.Errorf("encode recovery codes: %w", err)
	}
	middleware.Set(recoveryCodesSessionKey, string(encoded))
	return nil
}

// ConsumeRecoveryCodes returns and removes one-time recovery codes.
func ConsumeRecoveryCodes(c fiber.Ctx) ([]string, bool) {
	middleware := fibersession.FromContext(c)
	if middleware == nil {
		return nil, false
	}
	raw, _ := middleware.Get(recoveryCodesSessionKey).(string)
	middleware.Delete(recoveryCodesSessionKey)
	var codes []string
	if raw == "" || json.Unmarshal([]byte(raw), &codes) != nil || len(codes) == 0 {
		return nil, false
	}
	return codes, true
}

// CurrentID returns the opaque transport ID for the current server-side session.
// Callers must treat it as a credential and never persist or expose it directly.
func CurrentID(c fiber.Ctx) string {
	middleware := fibersession.FromContext(c)
	if middleware == nil {
		return ""
	}
	return middleware.ID()
}

// flashSessionPrefix namespaces one-time values that must survive exactly one
// redirect without being exposed in the URL.
const flashSessionPrefix = "flash:"

// SetFlash stores a one-time value in the server-side session.
func SetFlash(c fiber.Ctx, key, value string) error {
	middleware := fibersession.FromContext(c)
	if middleware == nil {
		return errors.New("session middleware is not available")
	}
	middleware.Set(flashSessionPrefix+key, value)
	return nil
}

// ConsumeFlash returns and removes a one-time value stored with SetFlash.
func ConsumeFlash(c fiber.Ctx, key string) (string, bool) {
	middleware := fibersession.FromContext(c)
	if middleware == nil {
		return "", false
	}
	value, _ := middleware.Get(flashSessionPrefix + key).(string)
	middleware.Delete(flashSessionPrefix + key)
	return value, value != ""
}
