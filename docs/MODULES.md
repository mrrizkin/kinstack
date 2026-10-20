# Modules & status

The KinStack base contains a generic foundation. Feature slices are added as new
vertical slices on top of this base.

## 1. Authentication (`auth`)

| Feature | Status |
|---|---|
| Username + password login/logout | Done |
| Break-glass super admin via `ACCESS_KEY`/`SECRET_KEY` (ephemeral, no DB account) | Done |
| Login and 2FA verification rate limits | Done |
| Per-account login lockout (atomic in Valkey) | Done |
| Valkey session store + session regeneration | Done |
| Two-step verification (TOTP) and cancellation | Done |
| Email magic link (one-time, 15-minute TTL) | Done |
| Self-service password reset (forgot password, one-time token with 60-minute TTL) | Done |
| User invitation (admin creates the account, 7-day invitation link, empty initial password) | Done |
| Passkey/WebAuthn (enroll, *discoverable* sign-in, revoke) | Done |
| Optional multi-provider SSO (OIDC + Google/Apple/Facebook/GitHub + SAML 2.0, PKCE, HMAC state, verified email) | Done |
| Per-method feature flags (`AUTH_*_ENABLED`), panic when enabled but the setting is invalid | Done |
| Personal API tokens (bearer, SHA-256 hash, expiry, revoke, separate permission scopes) | Done |

## 2. Users & authorization (`users`, `platform/authz`)

| Feature | Status |
|---|---|
| User CRUD (create, update, activate/deactivate, reset password) | Done |
| New user invitation via email (`POST /users/invite`, `users.manage`) | Done |
| Custom roles (create, update, delete, set permissions) | Done |
| Data-driven roles: no role names in code, permissions come purely from the DB | Done |
| Hybrid RBAC + ABAC permissions | Done |
| Per-user permission overrides (allow/deny, deny wins) | Done |
| *Fail-closed* route guard (`authz.GuardRouter`) | Done |

The permission registry is owned by Go in
`internal/platform/authz/catalog.go`. The base registers the `users.view`,
`users.manage`, `access.manage`, `audit.view`, `media.view`, and `media.manage`
permissions. Every new slice adds its own permission codes to this registry.

## 3. Profile & preferences (`profile`)

| Feature | Status |
|---|---|
| Display name | Done |
| Change password | Done |
| Two-step verification & recovery codes | Done |
| Active sessions (view & revoke) | Done |
| Theme, notification sound | Done |

## 4. Audit (`platform/audit` + `features/audit`)

| Feature | Status |
|---|---|
| Append-only audit event for every HTTP mutation | Done |
| Audit viewer (`GET /audit`): actor, action, outcome, date, search, and pagination filters | Done |

## 5. Media (`platform/objectstorage` + `features/media`)

| Feature | Status |
|---|---|
| Upload files to S3-compatible object storage (RustFS), max 10 MB; images validated by magic bytes and a 12,000 px limit | Done |
| List assets per owner, serve private files through authenticated routes | Done |
| Delete assets (storage + soft-delete metadata) | Done |
| `MEDIA_ENABLED` feature flag; an invalid RustFS setting while enabled causes a panic | Done |

## 6. Notifications (`features/notifications`)

| Feature | Status |
|---|---|
| In-app notification center: list, mark read, mark all read | Done |
| `Service.Notify` so other slices can send notifications to users | Done |

## 7. System surfaces (`system`)

| Feature | Status |
|---|---|
| Public 403/error/offline/maintenance pages | Done |
| Help page & keyboard shortcuts | Done |
| CSP report endpoint (`POST /csp-report`) | Done |

## 8. Observability & external integrations (`platform/observability`)

| Feature | Status |
|---|---|
| GlitchTip error tracking server (Sentry SDK, request/trace tags) | Done |
| GlitchTip browser (Vue error handler, lazy-loaded) | Done |
| OpenObserve OTLP/HTTP traces + W3C propagation | Done |
| OpenObserve OTLP/HTTP logs from zerolog | Done |
| OpenObserve HTTP metrics (request count + latency) | Done |
| Umami analytics (self-hosted script, SPA auto-track) | Done |
| Renovate for Forgejo/GitHub (`renovate.json`) | Done |
| Graceful shutdown + telemetry flush | Done |
| CSP nonce (inline scripts, Vite, Inertia) + Umami/GlitchTip sources | Done |

All integrations are opt-in (`GLITCHTIP_DSN`, `OPENOBSERVE_ENABLED`,
`UMAMI_ENABLED`) and are described in [`OBSERVABILITY.md`](./OBSERVABILITY.md).

## 9. Email (`platform/email`)

| Feature | Status |
|---|---|
| Provider-neutral boundary (SMTP relay) + `UnavailableSender`/`MemorySender` | Done |
| Maizzle templates (`resources/js/emails`, built to `assets/emails/dist`) | Done |
| Mailpit for development | Done |

Email uses `[[.Field]]` as the runtime delimiter because `{{ }}` is consumed by
Vue at build time. Leave `SMTP_HOST` empty to disable delivery.

## Route map

| Route | Slice | Access |
|---|---|---|
| `GET /dashboard` | `dashboard` | authenticated |
| `GET /login`, `POST /login`, `GET /2fa`, `POST /2fa`, `POST /2fa/cancel`, `POST /logout` | `auth` | public |
| `GET/POST /login/forgot`, `GET/POST /login/reset/:token` | `auth` | public |
| `GET /profile`, `POST /profile`, `POST /profile/preferences`, `POST /profile/password`, `POST /profile/sessions/:publicID/revoke`, `POST /profile/sessions/revoke-others`, `POST /profile/two-factor/{begin,confirm,disable,recovery-codes}` | `profile` | authenticated |
| `GET/POST/PUT /users…`, `POST /users/:publicID/{password,active}` | `users` | `users.view` / `users.manage` |
| `GET/POST/PUT /access…` | `users` | `access.manage` |
| `GET /audit` | `audit` | `audit.view` |
| `GET /media`, `GET /media/:publicID/file` | `media` | `media.view` |
| `POST /media`, `POST /media/:publicID/delete` | `media` | `media.manage` |
| `GET /notifications`, `POST /notifications/read-all`, `POST /notifications/:publicID/read` | `notifications` | authenticated |
| `GET /{403,error,offline,maintenance,help,help/shortcuts}`, `GET /404` | `system` | public |
| `POST /csp-report` | `system` | public |

Routes live at the application root (no prefix).
The access column above is the baseline; every route is guarded by a
*fail-closed* permission through `routePolicy` in `bootstrap/web_routes.go`.
