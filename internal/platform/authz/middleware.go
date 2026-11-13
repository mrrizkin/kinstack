package authz

import (
	"kinstack/internal/pkg/assert"
	"kinstack/internal/platform/actor"
	"kinstack/internal/platform/i18n"

	"github.com/gofiber/fiber/v3"
)

const decisionLocalKey = "authz_decision"

// RoutePolicy maps "METHOD /path" to the permission required by that route.
// Every route registered through a GuardRouter must appear here.
type RoutePolicy map[string]Permission

// ResourceResolver loads the attributes an ABAC policy needs from the request.
// It must be cheap and side-effect free; returning nil means "no attributes".
type ResourceResolver func(fiber.Ctx) Resource

// GuardRouter keeps bootstrap/web_routes.go the single place that decides who may
// do what, without threading an authorizer through every handler.
type GuardRouter struct {
	fiber.Router

	actor       *actor.Actor
	service     *Service
	policy      RoutePolicy
	resolvers   map[string]ResourceResolver
	middlewares []fiber.Handler
}

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

func (g *GuardRouter) WithResolvers(resolvers map[string]ResourceResolver) *GuardRouter {
	g.resolvers = resolvers
	return g
}

func (g *GuardRouter) WithMiddleware(middlewares ...fiber.Handler) *GuardRouter {
	g.middlewares = append(g.middlewares, middlewares...)
	return g
}

func (g *GuardRouter) Get(path string, handler any, handlers ...any) fiber.Router {
	return register(g, fiber.MethodGet, path, handler, handlers)
}

func (g *GuardRouter) Post(path string, handler any, handlers ...any) fiber.Router {
	return register(g, fiber.MethodPost, path, handler, handlers)
}

func (g *GuardRouter) Put(path string, handler any, handlers ...any) fiber.Router {
	return register(g, fiber.MethodPut, path, handler, handlers)
}

func (g *GuardRouter) Delete(path string, handler any, handlers ...any) fiber.Router {
	return register(g, fiber.MethodDelete, path, handler, handlers)
}

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

// RequireBrowserSession rejects personal API tokens on credential management
// routes: a leaked token must not mint new credentials.
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

// RequirePermissionResource runs after the handler loads the target entity.
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

func DecisionFromContext(c fiber.Ctx) (Decision, bool) {
	decision, ok := c.Locals(decisionLocalKey).(Decision)
	return decision, ok
}
