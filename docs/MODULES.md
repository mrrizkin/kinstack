# Modules

KinStack is organized as vertical slices under `internal/features`. Each slice
owns its HTTP boundary, application logic, repositories, errors, and DTOs.

## Authentication

`auth` owns username/password login, break-glass super admin access, rate limits,
per-account lockout, Valkey sessions, two-step verification, magic links,
password reset, invitations, passkeys, OAuth/OIDC, SAML, and API token
authentication.

Authentication methods are configured with `AUTH_*` environment variables. An
enabled method with incomplete settings fails at startup.

## Users and authorization

`users` owns user records, invitations, roles, and access management.
`platform/authz` owns the permission registry, hybrid RBAC + ABAC evaluation,
per-user overrides, and the fail-closed route guard.

Permission codes are defined in `internal/platform/authz/catalog.go`. The base
registers `users.view`, `users.manage`, `access.manage`, `audit.view`,
`media.view`, and `media.manage`.

## Profile

`profile` owns display names, password changes, themes, notification sound,
active browser sessions, two-step verification, and recovery codes.

## Audit

`platform/audit` records append-only events for HTTP mutations and selected
reads. `features/audit` provides the authenticated audit viewer with actor,
action, outcome, date, search, and pagination filters.

## Media

`features/media` stores files in S3-compatible object storage behind
`MEDIA_ENABLED`. Uploads are limited to 10 MiB, validated by magic bytes, and
images are limited to 12,000 px per side. Files are served through authenticated
routes controlled by `media.view`.

## Notifications

`features/notifications` provides the in-app notification center, read state, and
the `Service.Notify` boundary used by other slices.

## System surfaces

`features/system` serves public error, offline, and maintenance surfaces, the
help page, keyboard shortcuts, and the public CSP report endpoint.

## Observability

`platform/observability` integrates GlitchTip, OpenObserve OTLP traces/logs/
metrics, and graceful telemetry shutdown. `resources/js/lib/umami.ts` integrates
Umami analytics in the browser.

All integrations are opt-in and have no effect when disabled.

## Email

`platform/email` provides a provider-neutral SMTP sender with Maizzle templates
compiled from `resources/js/emails` into `assets/emails/dist`.

Runtime templates use `[[.Field]]` delimiters because `{{ }}` is consumed by Vue
at build time. An empty `SMTP_HOST` disables delivery.

## Route map

| Route | Slice | Access |
|---|---|---|
| `/login`, `/login/forgot`, `/login/reset/:token`, `/2fa`, `/logout` | `auth` | public |
| `/dashboard` | `dashboard` | authenticated |
| `/profile`, `/profile/preferences`, `/profile/password`, `/profile/sessions/*`, `/profile/two-factor/*` | `profile` | authenticated |
| `/api-tokens` | `apitoken` | authenticated |
| `/passkeys` | `passkeys` | authenticated |
| `/users`, `/users/invite`, `/users/:publicID/*` | `users` | `users.view` / `users.manage` |
| `/access`, `/access/roles/*`, `/access/users/*` | `users` | `access.manage` |
| `/audit` | `audit` | `audit.view` |
| `/media`, `/media/:publicID/file` | `media` | `media.view` |
| `/media` POST, `/media/:publicID/delete` | `media` | `media.manage` |
| `/notifications`, `/notifications/feed`, `/notifications/*` | `notifications` | authenticated |
| `/help`, `/403`, `/error`, `/offline`, `/maintenance`, `/404` | `system` | public |
| `/csp-report` | `system` | public |

Routes are registered at the application root. Every workspace route has a
matching entry in `routePolicy` in `internal/bootstrap/web_routes.go`.
