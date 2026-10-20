// Package passkeys owns WebAuthn registration, discoverable sign-in, and
// actor-scoped credential management.
package passkeys

import (
	"net/url"
	"strings"

	"kinstack/internal/features/auth"
	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	platformactor "kinstack/internal/platform/actor"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gofiber/fiber/v3"
)

// Dependency contains the persistence, WebAuthn, authentication, actor, and
// logging boundaries used by the feature.
type Dependency struct {
	Repository Repository
	AppURL     string
	Auth       *auth.Feature
	Actor      *platformactor.Actor
	Logger     applog.Log
}

// Feature owns the passkey service and HTTP handler.
type Feature struct {
	service *Service
	handler *Handler
}

// New constructs the passkey feature from explicit dependencies.
func New(d Dependency) *Feature {
	assert.Assert(d.Repository != nil, "passkeys repository is required")
	assert.Assert(d.Auth != nil, "passkeys auth feature is required")
	assert.Assert(d.Actor != nil, "passkeys actor boundary is required")
	assert.Assert(d.Logger != nil, "passkeys logger is required")
	assert.Assert(d.AppURL != "", "passkeys app URL is required")

	parsed, err := url.Parse(strings.TrimSpace(d.AppURL))
	assert.Assert(
		err == nil && parsed.Hostname() != "" && parsed.Scheme != "",
		"passkeys app URL is invalid",
	)
	webAuthn := assert.OK(newWebAuthn(parsed.Hostname(), parsed.Scheme+"://"+parsed.Host))

	service := NewService(ServiceDependency{Repository: d.Repository})
	return &Feature{
		service: service,
		handler: NewHandler(HandlerDependency{
			Actor:    d.Actor,
			Service:  service,
			WebAuthn: webAuthn,
			Auth:     d.Auth,
			Logger:   d.Logger,
		}),
	}
}

// Mount registers the public passkey sign-in routes.
func (f *Feature) Mount(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "passkeys feature must be initialized")
	f.handler.MountPublic(router)
}

// MountWorkspace registers the authenticated credential-management routes.
func (f *Feature) MountWorkspace(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "passkeys feature must be initialized")
	f.handler.MountWorkspace(router)
}

// Service exposes the feature service for composition-root adapters.
func (f *Feature) Service() *Service {
	assert.Assert(f != nil && f.service != nil, "passkeys feature must be initialized")
	return f.service
}

func newWebAuthn(rpID, origin string) (*webauthn.WebAuthn, error) {
	return webauthn.New(
		&webauthn.Config{RPDisplayName: "KinStack", RPID: rpID, RPOrigins: []string{origin}},
	)
}
