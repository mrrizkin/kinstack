# KinStack

KinStack is a generic application base built with Go, Fiber v3, Inertia, Vue 3,
Nuxt UI, PostgreSQL, and Valkey. It provides the shared platform concerns that
most internal applications need before domain-specific work begins.

## Capabilities

- Username and password authentication with Argon2id, optional TOTP two-step
  verification, passkeys, magic links, password reset, invitations, OAuth/OIDC,
  SAML, and break-glass super admin credentials.
- User, role, and permission management with hybrid RBAC and ABAC, including
  per-user allow/deny overrides and fail-closed route policies.
- Profile and account security: display name, password, active sessions, theme,
  notification sound, two-step verification, and personal API tokens.
- Append-only audit records for HTTP mutations and an audit viewer.
- S3-compatible media storage with authenticated file delivery and upload
  validation.
- In-app notifications, dashboard shell, public landing page, and split-screen
  authentication pages.
- Optional GlitchTip error tracking, OpenObserve telemetry, and Umami analytics.
- English-only i18n boundaries for frontend copy and Go response messages.

## Stack

| Layer | Technology |
|---|---|
| Backend | Go 1.27, Fiber v3 |
| Frontend | Vue 3, Inertia, Nuxt UI 4, Vite |
| Database | PostgreSQL, sqlx, pgx, Goose |
| Sessions and cache | Valkey |
| Object storage | S3-compatible storage (RustFS in development) |
| Documents | Gotenberg |
| Email | SMTP, Maizzle templates |
| Development tooling | Bun, `just`, Docker Compose |

## Requirements

- Go 1.27+
- Bun
- Docker and Docker Compose
- `just`
- `ripgrep` (`rg`) for `just check-feature-slices`

## Quick start

```bash
cp .env.example .env
just setup
just infra-up
just migrate
just dev
```

Open <http://localhost:3000>. Mailpit captures development email at
<http://localhost:8025>.

`just infra-up` starts the core services and the local observability stack. To
start only the services required by `just dev`, use `just infra-up-core`.

## Commands

```bash
just dev                 # application + Vite dev server
just test                # types, Go tests, frontend tests
just verify              # full gate: build, tests, vet, staticcheck, lint, format
just audit               # frontend audit + govulncheck
just coverage            # informational Go coverage report
just build               # production assets + binary
just migrate             # apply Goose migrations
just migrate-status      # show the applied migration
just smoke-csp           # CSP smoke test with a production build
just infra-up            # core services + GlitchTip, Umami, OpenObserve
just infra-up-core       # PostgreSQL, Valkey, Mailpit, RustFS, Gotenberg
just infra-down          # stop the local stack
```

## Local observability

| Tool | URL |
|---|---|
| GlitchTip | <http://localhost:8000> |
| Umami | <http://localhost:3003> |
| OpenObserve | <http://localhost:5080> |

The application integrations remain opt-in. Create the resources in each UI,
then set the corresponding environment variables:

- `GLITCHTIP_DSN` from a GlitchTip project.
- `UMAMI_ENABLED=true`, `UMAMI_SCRIPT_URL=http://localhost:3003/script.js`, and
  `UMAMI_WEBSITE_ID` from an Umami website.
- `OPENOBSERVE_ENABLED=true` and the local `OPENOBSERVE_AUTH_HEADER` documented
  in `.env.example`.

## Super admin

Set `ACCESS_KEY` and `SECRET_KEY` in `.env`. Both are required together, and
`SECRET_KEY` must contain at least 32 characters. These credentials are checked
before the database, do not create an account, and grant an ephemeral session
that bypasses RBAC.

Use the break-glass session to create the first roles and users from the
**Users** and **Access** pages. The session has no database profile, so account
pages such as profile and passkeys are not available to it.

## Routes

| Route | Access |
|---|---|
| `/login`, `/2fa`, `/logout`, `/login/forgot`, `/login/reset/:token` | public |
| `/dashboard` | authenticated |
| `/profile`, `/api-tokens`, `/passkeys`, `/notifications` | authenticated |
| `/users` | `users.view` / `users.manage` |
| `/access` | `access.manage` |
| `/audit` | `audit.view` |
| `/media` | `media.view` / `media.manage` |
| `/help`, `/403`, `/error`, `/offline`, `/maintenance` | public |

## Configuration

Configuration is read from environment variables and validated at startup.
`.env.example` documents every setting. Production requires HTTPS `APP_URL`,
secure cookies, CSP, and non-placeholder secrets.

## Documentation

- [Architecture](./docs/ARCHITECTURE.md)
- [Modules](./docs/MODULES.md)
- [Operations](./docs/OPERATIONS.md)
- [Observability](./docs/OBSERVABILITY.md)
- [Internationalization](./docs/INTERNATIONALIZATION.md)
- [Renovate](./docs/RENOVATE.md)
- [Security policy](./SECURITY.md)

## License

[MIT](./LICENSE)
