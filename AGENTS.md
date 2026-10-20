# Agent Instructions

This repository is KinStack, a generic application base. Keep it domain-neutral.

## Commands

```bash
just setup          # install frontend dependencies
just dev            # application + Vite
just verify         # required before finishing a change
just test           # types, Go tests, frontend tests
just audit          # dependency audits
just migrate        # apply migrations
```

## Structure

- `internal/features/<slice>` contains vertical slices with canonical files:
  `handler.go`, `service.go`, `repository.go`, `repository_pg.go`,
  `repository_mem.go`, `dto.go`, `errors.go`, and tests.
- `internal/platform` contains shared infrastructure.
- `config` contains validated environment configuration.
- `resources/js` contains Vue pages, layouts, components, i18n, and email
  templates.
- `db/migrations` contains Goose migrations.
- `deploy/compose.yml` contains local and deployment services.

## Conventions

- Keep application logic in `service.go`. Handlers parse, validate transport
  input, and map errors.
- Repository interfaces belong to the consuming feature.
- DTOs use `snake_case` JSON without `db` tags.
- Register permission codes in `internal/platform/authz/catalog.go`.
- Register every guarded route in `routePolicy` in
  `internal/bootstrap/web_routes.go`.
- Use the Go response catalog in `internal/platform/i18n` for user-facing
  messages.
- Use `vue-i18n` keys for shared frontend chrome. Page copy is English.
- Do not introduce product-specific modules, terminology, or examples. KinStack
  is a base, not an application domain.
- Do not commit `.env`, secrets, build output, `node_modules`, or local state.

## UI

- Use Nuxt UI components and the existing shell patterns.
- Import local components explicitly.
- Use `AppInputDate` for date inputs, `AppTableCard` for desktop tables, and
  `AppMobileList` for mobile lists.
- Keep the English-only locale and `en-US` formatters unless the task explicitly
  adds a locale.

## Verification

Run `just verify` before reporting completion. For CSP or production behavior,
also run `just smoke-csp`. Report checks that could not run and why.
