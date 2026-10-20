// Package auth owns username/password sign-in and sign-out for the application.
package auth

import (
	"time"

	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/email"
	platformoauth "kinstack/internal/platform/oauth"
	platformsaml "kinstack/internal/platform/saml"
	"kinstack/internal/platform/sessionregistry"
	"kinstack/internal/platform/twofactor"

	"github.com/gofiber/fiber/v3"
)

// Dependency contains the repository, actor boundary, environment super admin
// credentials, outbound-email boundary, session registry, two-factor service,
// and logger used by the feature.
type Dependency struct {
	Repository    Repository
	Actor         *actor.Actor
	Logger        applog.Log
	AccessKey     string
	SecretKey     string
	Email         email.Sender
	EmailFrom     string
	BaseURL       string
	Storage       fiber.Storage
	LockoutMax    int
	LockoutWindow time.Duration
	Sessions      *sessionregistry.Service
	TwoFactor     *twofactor.Service

	MagicLinkEnabled bool
	PasskeyEnabled   bool
	PasswordReset    bool

	OAuthProviders   []platformoauth.Provider
	OAuthStateSecret string
	OAuthRedirectURL string

	SAMLProvider *platformsaml.Provider
	SAMLACSURL   string
}

// Feature owns authentication service and HTTP entry points.
type Feature struct {
	service *Service
	handler *Handler
}

// New constructs the auth feature from explicit dependencies.
func New(d Dependency) *Feature {
	assert.Assert(d.Repository != nil, "auth repository is required")
	assert.Assert(d.Actor != nil, "auth actor boundary is required")
	assert.Assert(d.Logger != nil, "auth logger is required")

	service := NewService(ServiceDependency{
		Repository: d.Repository,
		AccessKey:  d.AccessKey,
		SecretKey:  d.SecretKey,
		Email:      d.Email,
		EmailFrom:  d.EmailFrom,
		BaseURL:    d.BaseURL,
		Lockout:    NewLockout(d.Storage, d.LockoutMax, d.LockoutWindow),
	})
	handler := NewHandler(HandlerDependency{
		Service:   service,
		Actor:     d.Actor,
		Logger:    d.Logger,
		Sessions:  d.Sessions,
		TwoFactor: d.TwoFactor,

		MagicLinkEnabled: d.MagicLinkEnabled,
		PasskeyEnabled:   d.PasskeyEnabled,
		PasswordReset:    d.PasswordReset,

		OAuthProviders:   d.OAuthProviders,
		OAuthStateSecret: d.OAuthStateSecret,
		OAuthRedirectURL: d.OAuthRedirectURL,

		SAMLProvider: d.SAMLProvider,
		SAMLACSURL:   d.SAMLACSURL,

		ContinuationStorage: d.Storage,
	})
	return &Feature{service: service, handler: handler}
}

// Mount registers the authentication routes.
func (f *Feature) Mount(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "auth feature must be initialized")
	f.handler.Mount(router)
}

// Service exposes the feature service for composition-root adapters such as the
// API bearer-token middleware.
func (f *Feature) Service() *Service {
	assert.Assert(f != nil && f.service != nil, "auth feature must be initialized")
	return f.service
}

// SetSession establishes the browser session for an identity that a non-password
// authenticator (such as a passkey) has already verified. It does not redirect.
func (f *Feature) SetSession(c fiber.Ctx, session SessionResponse) error {
	assert.Assert(f != nil && f.handler != nil, "auth feature must be initialized")
	return f.handler.setSession(c, session)
}
