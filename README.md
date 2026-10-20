# KinStack

Application base built with Go + Fiber v3 + Inertia (Vue 3 + Nuxt UI), PostgreSQL
(sqlx + Goose), and Valkey. This repository is a **generic foundation**:
authentication, user management, RBAC + ABAC authorization, profiles, account
security, and a dashboard shell. Feature slices are not included and are added
on top of this base as separate vertical slices.

## What the base includes

| Area | Status |
|---|---|
| Username + password auth (break-glass super admin via env) | Done |
| User & role management | Done |
| Hybrid RBAC + ABAC authorization (roles, permissions, per-user overrides) | Done |
| Profile & preferences (theme, sound, active sessions, 2FA) | Done |
| Data change audit | Done |
| Dashboard shell (Nuxt UI `dashboard-vue`) | Done |
| Opt-in observability: GlitchTip, OpenObserve (OTLP), Umami | Done |
| Feature slices | Not included |

See [`docs/README.md`](./docs/README.md) for the module map,
[`docs/ARCHITECTURE.md`](./docs/ARCHITECTURE.md) for structure and rules,
[`docs/OBSERVABILITY.md`](./docs/OBSERVABILITY.md) for the GlitchTip,
OpenObserve, and Umami integrations, and
[`docs/INTERNATIONALIZATION.md`](./docs/INTERNATIONALIZATION.md) for the
frontend/backend i18n boundary. The backup/alert/incident runbook lives in
[`docs/OPERATIONS.md`](./docs/OPERATIONS.md); the accessibility checklist is in
[`docs/ACCESSIBILITY.md`](./docs/ACCESSIBILITY.md).

## Development

Requirements: Go 1.27+, Bun, Docker, [`just`](https://just.systems/), and
`ripgrep` (`rg`) for `just check-feature-slices`.

```bash
cp .env.example .env
just setup              # bun install
just infra-up           # PostgreSQL + Valkey + Mailpit + observability UIs
just migrate            # create the schema
just dev                # app + vite dev server
```

Open <http://localhost:3000>. Development email is captured by Mailpit at
<http://localhost:8025> (`SMTP_HOST=localhost`, `SMTP_PORT=1025`).

Maizzle email preview runs at <http://localhost:3001> while Vite dev is running.

### Super admin login

Login uses credentials from the environment, not a database account:

- `ACCESS_KEY` = username
- `SECRET_KEY` = password

Set both in `.env` (generate them yourself, at least 32 characters). These
credentials are a break-glass super admin that works like HTTP basic auth: they
are checked before the database, do not create or promote any account, and the
session is ephemeral with full access that bypasses RBAC. After signing in,
create other roles and accounts from the **Users** and **Access** menus.

After login, the workspace opens the **Dashboard** page. The dashboard contains
quick access to the pages available under the current permissions, plus
shortcuts to account settings. The break-glass super admin is also sent to the
Dashboard, but profile settings, passkeys, and API tokens are unavailable
because that account has no database profile.

Routes use normal paths at the application root: `/login`, `/2fa`, `/logout`,
`/dashboard`, `/profile`, `/users`, and `/access`.

### Other commands

```bash
just test               # vue-tsc + go test + bun test
just coverage           # Go coverage report (optional)
just verify             # full gate: types, tests, vet, staticcheck, lint, format
just audit              # bun audit + govulncheck
just smoke-csp          # CSP smoke test with a production build (Vite must be stopped)
just check-feature-slices
just build              # build assets + binary
just infra-down
```

### Migrations during deployment

The schema is managed by Goose and embedded in the binary (`db/migrations`).
Apply it before or alongside a release:

```bash
just migrate        # same as: go run ./cmd/migrate up
just migrate-status # show the applied version
```

### Local hardening

- The default request body limit is 12 MiB (`APP_BODY_LIMIT_MB`), while media is
  capped at 10 MiB so multipart overhead does not cut off uploads.
- Media upload validates file contents (magic bytes), not just the browser
  `Content-Type`, and rejects images above 12,000 px per side.
- Login is rate limited per IP and locked out per account
  (`AUTH_LOCKOUT_MAX`, `AUTH_LOCKOUT_WINDOW`).
- Requests using an API bearer token are rate limited per token
  (`API_RATE_LIMIT_MAX`, `API_RATE_LIMIT_WINDOW`).
- The HTTP server uses `HTTP_READ_TIMEOUT`, `HTTP_WRITE_TIMEOUT`, and
  `HTTP_IDLE_TIMEOUT`.
- Application queries use `DB_STATEMENT_TIMEOUT` (default 15 seconds);
  migrations are not subject to this limit.
- CSP can be tested with `just smoke-csp` after `bun x vite build`.

The security policy and reporting process are in
[`SECURITY.md`](./SECURITY.md). The license is in [`LICENSE`](./LICENSE).

### Dependency scanning

```bash
bun audit                                             # frontend dependencies
go run golang.org/x/vuln/cmd/govulncheck@latest ./... # backend dependencies
```

Renovate is configured in `renovate.json` for Go modules, Bun/npm, GitHub
Actions, Dockerfile, and container images (`deploy/compose.yml`). See
[`docs/RENOVATE.md`](./docs/RENOVATE.md) for running it on Forgejo.

### Container deployment

`Containerfile` builds the frontend assets (Bun) and the Go binary + `migrate`,
then runs them in a distroless image. On every push to `main`, the
`docker-build.yml` workflow publishes the image to GitHub Container Registry with
an automatic version tag (`v<YEAR>.<WEEK>.<PATCH>`) and `latest`.

## Local observability stack

`just infra-up` starts the core services plus the optional observability tools:

| Tool | UI |
|---|---|
| GlitchTip | <http://localhost:8000> |
| Umami | <http://localhost:3003> |
| OpenObserve | <http://localhost:5080> |

Start the stack, then create the resources for each tool. The application
integrations remain opt-in:

- **GlitchTip**: create a project in the GlitchTip UI and set
  `GLITCHTIP_DSN` to the project DSN.
- **Umami**: create a website in the Umami UI and set
  `UMAMI_ENABLED=true`, `UMAMI_SCRIPT_URL=http://localhost:3003/script.js`, and
  `UMAMI_WEBSITE_ID=<id>`.
- **OpenObserve**: set `OPENOBSERVE_ENABLED=true` and
  `OPENOBSERVE_AUTH_HEADER="Basic cm9vdEBraW5zdGFjay5sb2NhbDpLaW5TdGFjayEyMDI2"` for
  the local OpenObserve login.

See [`docs/OBSERVABILITY.md`](./docs/OBSERVABILITY.md) for the full integration
details.
