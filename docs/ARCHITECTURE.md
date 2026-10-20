# Architecture

KinStack is a modular monolith. Each capability is a vertical slice under
`internal/features`, with shared infrastructure under `internal/platform`.

```
HTTP (Fiber)
  └── global middleware (helmet, recover, request ID, logging, session, audit, CSRF, Inertia)
        └── bootstrap/web_routes.go
              └── internal/features/<slice>
                    handler.go        HTTP boundary: parse, validate, map errors
                    service.go        application logic
                    repository.go     consumer-owned repository interface
                    repository_pg.go  PostgreSQL adapter
                    repository_mem.go in-memory test adapter
                    dto.go            request/response types
                    errors.go         expected application errors
```

## Platform boundaries

| Package | Responsibility |
|---|---|
| `platform/actor` | session actor and account validation |
| `platform/authz` | RBAC + ABAC, permission registry, overrides, route guard |
| `platform/audit` | append-only audit events |
| `platform/database` | pgx connection, pool settings, prepared statements, Goose |
| `platform/email` | SMTP delivery and Maizzle template rendering |
| `platform/i18n` | English response-message catalog |
| `platform/observability` | GlitchTip, OpenObserve OTLP, telemetry shutdown |
| `platform/oauth` | OAuth 2.0 / OIDC providers |
| `platform/saml` | SAML 2.0 service provider |
| `platform/inertia` | Fibertia adapter and document shell |
| `platform/pdf` | Gotenberg HTML-to-PDF client |
| `platform/password` | Argon2id hashing |
| `platform/session` | Valkey session store and 2FA challenge state |
| `platform/sessionregistry` | active browser session registry |
| `platform/totp` | TOTP generation and verification |
| `platform/twofactor` | 2FA enrollment, confirmation, disable, recovery codes |

## Rules

- Application logic belongs in `service.go`; handlers do not make business
  decisions.
- Repository interfaces are owned by the consuming feature.
- DTOs use `snake_case` JSON and never expose database tags.
- Expected errors are typed and mapped at the HTTP boundary. Unexpected errors
  are logged with `request_id` and `error`.
- Permission codes live in `internal/platform/authz/catalog.go`.
- Every guarded route has an entry in `routePolicy` in
  `internal/bootstrap/web_routes.go`. Missing entries fail during wiring.
- `scripts/check-feature-slices.sh` enforces the canonical slice shape.

## UI

- The dashboard shell uses Nuxt UI `dashboard-vue`: sidebar, workspace menu,
  command palette, notification slide-over, user menu, and light/dark mode.
- Navigation groups are `Main`, `Administration`, and `System`. Items are
  filtered by permission; empty groups are hidden.
- Pages use `UDashboardPanel` and `UDashboardNavbar`.
- Public pages use `PublicLayout`; authentication pages use `AuthLayout`.
- Shared local components are imported explicitly; Nuxt UI components are
  auto-imported.
- Tables use `AppTableCard`, `AppTableTools`, and `UTable` on desktop. Mobile
  pages use `AppMobileList`.
- Form controls use Nuxt UI primitives. `AppInputDate` wraps `UInputDate` with a
  calendar popover and `YYYY-MM-DD` values.
- Server flash messages are shown through `AppFlashToaster`. Destructive actions
  use `AppConfirmDialog`.
- Locale is English. Dates and numbers use `en-US`; currency remains IDR.

## Adding a vertical slice

1. Create `internal/features/<slice>` with the canonical files.
2. Add permission codes in `internal/platform/authz/catalog.go`.
3. Register routes in `routePolicy` and mount the feature in
   `internal/bootstrap/app.go` and `web_routes.go`.
4. Add a Goose migration in `db/migrations`.
5. Add the Vue page under `resources/js/pages/<slice>/` and a navigation entry.
6. Run `just verify`.
