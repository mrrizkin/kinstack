// Package passkeys owns WebAuthn registration, discoverable sign-in, and
// actor-scoped credential management.
package passkeys

import (
	"net/url"
	"strings"

	"kinstack/internal/pkg/assert"
	applog "kinstack/internal/pkg/log"
	"kinstack/internal/pkg/pagination"
	platformactor "kinstack/internal/platform/actor"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gofiber/fiber/v3"
)

type Dependency struct {
	Repository Repository
	AppURL     string
	Sessions   SessionStarter
	Actor      *platformactor.Actor
	PageSize   pagination.PageSizeReader
	Logger     applog.Log
}

// Feature owns the passkey service and HTTP handler.
type Feature struct {
	service *Service
	handler *Handler
}

func New(d Dependency) *Feature {
	assert.Assert(d.Repository != nil, "passkeys repository is required")
	assert.Assert(d.Sessions != nil, "passkeys session starter is required")
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
			Sessions: d.Sessions,
			PageSize: d.PageSize,
			Logger:   d.Logger,
		}),
	}
}

func (f *Feature) Mount(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "passkeys feature must be initialized")
	f.handler.MountPublic(router)
}

// MountWorkspace registers the authenticated credential-management routes.
func (f *Feature) MountWorkspace(router fiber.Router) {
	assert.Assert(f != nil && f.handler != nil, "passkeys feature must be initialized")
	f.handler.MountWorkspace(router)
}

func (f *Feature) Service() *Service {
	assert.Assert(f != nil && f.service != nil, "passkeys feature must be initialized")
	return f.service
}

func newWebAuthn(rpID, origin string) (*webauthn.WebAuthn, error) {
	return webauthn.New(
		&webauthn.Config{RPDisplayName: "KinStack", RPID: rpID, RPOrigins: []string{origin}},
	)
}
