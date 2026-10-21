# KinStack

Go · Fiber v3 · Inertia · Vue 3 · Nuxt UI · PostgreSQL · Valkey

## Requirements

- Go 1.27+
- Bun
- Docker and Docker Compose
- `just`
- `ripgrep`

## Development

```bash
cp .env.example .env
just setup
just infra-up
just migrate
just dev
```

Open <http://localhost:3000>. Development email is captured at
<http://localhost:8025>.

## Commands

```bash
just dev                 # application + Vite
just test                # types, Go tests, frontend tests
just verify              # build, tests, vet, staticcheck, lint, format
just audit               # frontend audit + govulncheck
just coverage            # informational Go coverage report
just build               # production assets + binary
just migrate             # apply Goose migrations
just migrate-status      # show the applied migration
just smoke-csp           # CSP smoke test with a production build
just infra-up            # local infrastructure from deploy/compose.yml
just infra-up-core       # core infrastructure only
just infra-down          # stop local infrastructure
```

## Layout

```text
cmd/                    application and migration binaries
config/                 environment configuration and validation
internal/bootstrap/     application wiring, routes, and middleware
internal/features/      vertical slices
internal/platform/      shared infrastructure
resources/js/           Vue pages, layouts, components, emails, and i18n
resources/views/        document shell
db/migrations/          Goose migrations
deploy/                 Docker Compose
assets/                 embedded build assets
```

## Documentation

- [Architecture](./docs/ARCHITECTURE.md)
- [Style](./docs/STYLE.md)
- [Operations](./docs/OPERATIONS.md)
- [Agent instructions](./AGENTS.md)
- [Security policy](./SECURITY.md)
- [License](./LICENSE)
