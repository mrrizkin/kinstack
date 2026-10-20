package authz

import (
	"kinstack/internal/pkg/assert"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/i18n"

	"github.com/gofiber/fiber/v3"
)

// decisionLocalKey is the context key holding the last authorization decision.
const decisionLocalKey = "authz_decision"

// RoutePolicy maps "METHOD /path" to the permission required by that route.
// Every route registered through a GuardRouter must appear here.
type RoutePolicy map[string]Permission

// ResourceResolver loads the attributes an ABAC policy needs from the request.
// It must be cheap and side-effect free; returning nil means "no attributes".
type ResourceResolver func(fiber.Ctx) Resource

// GuardRouter decorates a fiber.Router so each registered route carries the
// permission guard declared in a RoutePolicy. This keeps the route table in
// bootstrap/web_routes.go as the single place that decides who may do what,
// without threading an authorizer through every feature handler.
type GuardRouter struct {
	fiber.Router

	actor       *actor.Actor
	service     *Service
	policy      RoutePolicy
	resolvers   map[string]ResourceResolver
	middlewares []fiber.Handler
}

// NewGuardRouter wraps router with permission guards.
func NewGuardRouter(
	router fiber.Router,
	actorBoundary *actor.Actor,
	service *Service,
	policy RoutePolicy,
) *GuardRouter {
	assert.Assert(router != nil, "guarded router requires a delegate")
	assert.Assert(actorBoundary != nil, "guarded router requires an actor boundary")
	assert.Assert(service != nil, "guarded router requires an authorization service")

	return &GuardRouter{Router: router, actor: actorBoundary, service: service, policy: policy}
}

// WithResolvers attaches resource resolvers for routes that need ABAC
// attributes (for example ownership of a sale). It returns the same router so
// it can be chained during wiring.
func (g *GuardRouter) WithResolvers(resolvers map[string]ResourceResolver) *GuardRouter {
	g.resolvers = resolvers
	return g
}

// WithMiddleware prepends middleware that runs before the authorization guard
// on every guarded route (for example session validation). It returns the same
// router so it can be chained during wiring.
func (g *GuardRouter) WithMiddleware(middlewares ...fiber.Handler) *GuardRouter {
	g.middlewares = append(g.middlewares, middlewares...)
	return g
}

// Get registers a guarded GET route.
func (g *GuardRouter) Get(path string, handler any, handlers ...any) fiber.Router {
	return register(g, fiber.MethodGet, path, handler, handlers)
}

// Post registers a guarded POST route.
func (g *GuardRouter) Post(path string, handler any, handlers ...any) fiber.Router {
	return register(g, fiber.MethodPost, path, handler, handlers)
}

// Put registers a guarded PUT route.
func (g *GuardRouter) Put(path string, handler any, handlers ...any) fiber.Router {
	return register(g, fiber.MethodPut, path, handler, handlers)
}

// Delete registers a guarded DELETE route.
func (g *GuardRouter) Delete(path string, handler any, handlers ...any) fiber.Router {
	return register(g, fiber.MethodDelete, path, handler, handlers)
}

// Patch registers a guarded PATCH route.
func (g *GuardRouter) Patch(path string, handler any, handlers ...any) fiber.Router {
	return register(g, fiber.MethodPatch, path, handler, handlers)
}

func register(
	g *GuardRouter,
	method string,
	path string,
	handler any,
	handlers []any,
) fiber.Router {
	chain := g.wrap(method, path, handler, handlers)
	switch method {
	case fiber.MethodGet:
		return g.Router.Get(path, chain[0], chain[1:]...)
	case fiber.MethodPost:
		return g.Router.Post(path, chain[0], chain[1:]...)
	case fiber.MethodPut:
		return g.Router.Put(path, chain[0], chain[1:]...)
	case fiber.MethodDelete:
		return g.Router.Delete(path, chain[0], chain[1:]...)
	case fiber.MethodPatch:
		return g.Router.Patch(path, chain[0], chain[1:]...)
	default:
		assert.Assertf(false, "unsupported guarded method %s", method)
		return nil
	}
}

func (g *GuardRouter) wrap(
	method string,
	path string,
	handler any,
	handlers []any,
) []any {
	permission, ok := g.policy[method+" "+path]
	assert.Assertf(ok, "missing route policy for %s %s", method, path)

	chain := make([]any, 0, len(g.middlewares)+len(handlers)+2)
	for _, middleware := range g.middlewares {
		chain = append(chain, middleware)
	}

	if permission == Authenticated {
		chain = append(chain, handler)
		return append(chain, handlers...)
	}

	if permission == BrowserSession {
		chain = append(chain, RequireBrowserSession(g.actor), handler)
		return append(chain, handlers...)
	}

	var guard fiber.Handler
	if resolve, ok := g.resolvers[method+" "+path]; ok && resolve != nil {
		guard = RequirePermissionResource(g.actor, g.service, permission, resolve)
	} else {
		guard = RequirePermission(g.actor, g.service, permission)
	}
	chain = append(chain, guard, handler)
	return append(chain, handlers...)
}

// PrincipalFromActor projects a session actor onto an authorization principal.
func PrincipalFromActor(session *actor.SessionActor) Principal {
	if session == nil {
		return Principal{}
	}
	return Principal{
		PublicID:    session.PublicID,
		Role:        session.Role,
		DisplayName: session.DisplayName,
		Superuser:   session.Superuser,
		Scopes:      session.Scopes,
	}
}

// RequireBrowserSession rejects personal API tokens. Credential management
// routes (password, 2FA, passkeys, API token lifecycle, session revocation) are
// guarded with it so a leaked token cannot mint new credentials: token scopes
// never restrict the Authenticated sentinel, and a passkey registered by a
// token is persistent access that outlives token revocation.
func RequireBrowserSession(actorBoundary *actor.Actor) fiber.Handler {
	assert.Assert(actorBoundary != nil, "actor boundary is required")

	return func(c fiber.Ctx) error {
		session := actorBoundary.ActorFromContext(c)
		if session == nil || session.PublicID == "" {
			return fiber.ErrUnauthorized
		}
		if session.Scopes != nil {
			c.Locals(decisionLocalKey, Deny(
				"browser_session",
				i18n.T("authz.reason.browser_session_required"),
			))
			return fiber.ErrForbidden
		}
		return c.Next()
	}
}

// RequirePermission rejects requests whose actor lacks the permission. It is a
// route-level RBAC guard: no resource attributes are available at this stage.
func RequirePermission(
	actorBoundary *actor.Actor,
	service *Service,
	permission Permission,
) fiber.Handler {
	assert.Assert(actorBoundary != nil, "actor boundary is required")
	assert.Assert(service != nil, "authorization service is required")

	return func(c fiber.Ctx) error {
		principal := PrincipalFromActor(actorBoundary.ActorFromContext(c))
		if principal.PublicID == "" {
			return fiber.ErrUnauthorized
		}
		decision := service.Can(c.Context(), principal, permission, nil)
		if !decision.Allowed {
			c.Locals(decisionLocalKey, decision)
			return fiber.ErrForbidden
		}
		return c.Next()
	}
}

// RequirePermissionResource is the resource-aware variant used after a handler
// loads the target entity. resolve must be cheap and side-effect free.
func RequirePermissionResource(
	actorBoundary *actor.Actor,
	service *Service,
	permission Permission,
	resolve func(fiber.Ctx) Resource,
) fiber.Handler {
	assert.Assert(actorBoundary != nil, "actor boundary is required")
	assert.Assert(service != nil, "authorization service is required")
	assert.Assert(resolve != nil, "resource resolver is required")

	return func(c fiber.Ctx) error {
		principal := PrincipalFromActor(actorBoundary.ActorFromContext(c))
		if principal.PublicID == "" {
			return fiber.ErrUnauthorized
		}
		decision := service.Can(c.Context(), principal, permission, resolve(c))
		if !decision.Allowed {
			c.Locals(decisionLocalKey, decision)
			return fiber.ErrForbidden
		}
		return c.Next()
	}
}

// DecisionFromContext returns the decision recorded by the most recent failed
// guard, when available.
func DecisionFromContext(c fiber.Ctx) (Decision, bool) {
	decision, ok := c.Locals(decisionLocalKey).(Decision)
	return decision, ok
}
