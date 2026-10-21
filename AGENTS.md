# Agent Instructions

## Commands

```bash
just setup
just dev
just verify
just test
just audit
just migrate
```

Run `just verify` before finishing a change. Follow `docs/STYLE.md` for code
conventions and principles.

## Layout

- `internal/features/<name>` — vertical slices.
- `internal/platform` — shared infrastructure.
- `internal/bootstrap` — wiring, middleware, routes.
- `config` — validated environment configuration.
- `resources/js` — Vue pages, layouts, components, i18n, emails.
- `db/migrations` — Goose migrations.
- `deploy/compose.yml` — local and deployment services.

## Rules

- New application behavior belongs in `internal/features/<name>`.
- Shared infrastructure belongs in `internal/platform`.
- Follow the handler/service/repository/DTO boundaries.
- Register permission codes in `internal/platform/authz/catalog.go`.
- Register guarded routes in `routePolicy` in
  `internal/bootstrap/web_routes.go`.
- Reuse existing components and helpers; do not create duplicates.
- Use the existing i18n boundaries and English copy.
- Do not commit `.env`, secrets, build output, `node_modules`, or local state.

## Verification

Run `just verify`. Run `just smoke-csp` for production CSP changes. Report checks
that could not run and why.
