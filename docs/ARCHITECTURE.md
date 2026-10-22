# Architecture

KinStack uses a modular monolith with one vertical slice per application
capability.

```text
HTTP
  └── global middleware
        └── bootstrap/web_routes.go
              └── internal/features/<slice>
                    handler.go
                    service.go
                    repository.go
                    repository_pg.go
                    repository_mem.go
                    dto.go
                    errors.go
```

## Directories

| Path | Responsibility |
|---|---|
| `cmd/` | application and migration entry points |
| `config/` | environment loading and startup validation |
| `internal/bootstrap/` | Fiber, middleware, dependency wiring, route table |
| `internal/features/` | application capabilities |
| `internal/platform/` | shared infrastructure |
| `resources/js/` | Vue pages, layouts, components, i18n, email templates |
| `resources/views/` | server-rendered document shell |
| `db/migrations/` | Goose migrations |
| `deploy/` | Docker Compose services |

## Feature slices

Each slice owns its HTTP boundary, application logic, persistence contract, and
tests.

| File | Responsibility |
|---|---|
| `handler.go` | parse input, validate transport data, map errors |
| `service.go` | application logic |
| `repository.go` | consumer-owned persistence interface |
| `repository_pg.go` | PostgreSQL adapter |
| `repository_mem.go` | in-memory adapter for tests |
| `dto.go` | request and response types |
| `errors.go` | expected application errors |

Rules:

- Application logic stays in `service.go`.
- Handlers do not make application decisions.
- Repository interfaces belong to the consuming slice.
- DTOs use `snake_case` JSON without database tags.
- Expected errors are typed. Unexpected errors are logged with `request_id`
  and `error`.
- `scripts/check-feature-slices.sh` enforces the canonical slice layout.

## Authorization

Permission codes are defined in `internal/platform/authz/catalog.go`.

Every guarded route has an entry in `routePolicy` in
`internal/bootstrap/web_routes.go`. A missing entry fails during startup.

### Delegation rules

Role and permission management is delegated, not all-or-nothing. An actor may
never grant more than they hold:

- Authority changes require an interactive browser session; API tokens are
  rejected even with `access.manage`.
- The actor must hold `access.manage`.
- The target role's rank must not exceed the actor's role rank, and a role may
  not be re-ranked above it. `user_roles` rank is enforced in the platform
  service, not just displayed in the UI.
- Every permission of the target role must already be effective for the actor,
  so `users.manage` alone cannot hand out `access.manage`.
- Actor accounts cannot change their own role or overrides, and cannot modify an
  account that outranks them.
- Built-in (`is_system`) roles are immutable at the service and SQL layers;
  change their permission set with a migration.

Handlers attach the acting principal with `authz.WithPrincipal`; services read
it with `authz.PrincipalFromContext` and fail closed when it is missing.

Resolved permission sets are cached per instance for `authz.DefaultCacheTTL`
(15 seconds) and the cache is capped at 10k entries. Role and override
mutations invalidate the affected users immediately on the serving replica;
other replicas converge within the TTL.

External identities are bound by `(provider, subject)` in `user_identities`.
The first sign-in links the subject to the account that owns the verified
email; later sign-ins use the binding. A subject never rebinds to another
account, and the base never provisions accounts from a provider.

## UI

- Dashboard pages use the existing dashboard shell.
- Public pages use `PublicLayout`; authentication pages use `AuthLayout`.
- Nuxt UI components are auto-imported. Local components are imported
  explicitly from `resources/js/components`.
- Reuse the existing components and helpers before creating new ones.
- Tables use the existing table wrapper and tooling. Narrow viewports use the
  existing mobile list component.
- Page props cross the trust boundary through `usePageSchema` and the zod
  schemas in `resources/js/lib/contracts`; domain types are inferred from those
  schemas instead of being declared twice.
- Go responses use the message registry in `internal/platform/i18n`. Shared
  frontend chrome uses `vue-i18n`. Page copy is English.

## Adding a slice

1. Create `internal/features/<name>` with the canonical files.
2. Add permission codes in `internal/platform/authz/catalog.go`.
3. Register routes in `routePolicy`.
4. Mount the feature in `internal/bootstrap/app.go` and `web_routes.go`.
5. Add a Goose migration in `db/migrations`.
6. Add the Vue page under `resources/js/pages/<name>/`.
7. Run `just verify`.
