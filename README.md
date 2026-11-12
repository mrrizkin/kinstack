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

Go builds embed `assets/emails/dist`; `just` targets build it on demand.

## Commands

`just --list` prints every recipe with its comment. `PILLARS.md` names the gate
each one enforces; `just verify` is the run that must pass before a change lands.

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
- [Engineering pillars](./PILLARS.md)
- [Operations](./docs/OPERATIONS.md)
- [Agent instructions](./AGENTS.md)
- [Security policy](./SECURITY.md)
- [License](./LICENSE)
