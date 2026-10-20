setup:
    bun install

[parallel]
dev: dev-app dev-assets

dev-app:
    go tool air

dev-assets:
    bun x vite

generate-views:
    go tool templ generate

build: build-assets build-app

build-app: generate-views
    go build -trimpath -o ./build/kinstack ./cmd/kinstack

build-assets:
    bun x vite build

format:
    bun x oxfmt resources/
    just format-go

format-go:
    go tool goimports -w .
    go tool gofumpt -w .
    go tool golines -w .

# vite build generates auto-imports.d.ts + components.d.ts (gitignored) which
# vue-tsc needs after a fresh clone / in CI.
generate-types:
    bun x vite build

type-check: generate-types
    bun x vue-tsc --build

lint-check:
    bun x oxlint resources

lint-go:
    go vet ./...
    go tool staticcheck ./...

check-feature-slices:
    ./scripts/check-feature-slices.sh

# Requires a production build and no running Vite dev server.
smoke-csp:
    ./scripts/smoke-csp.sh

format-check:
    bun x oxfmt --check resources/

test: type-check test-go test-frontend

test-go:
    go test ./...

test-frontend:
    bun test resources/js

# Print a local Go coverage report. This is informational, not a CI gate.
coverage:
    go test ./... -coverprofile=/tmp/kinstack-cover.out
    @go tool cover -func=/tmp/kinstack-cover.out | tail -n 1

# Full gate before committing: types, tests, lint, format, feature slices.
verify: type-check test-go test-frontend lint-go lint-check format-check check-feature-slices

# Structural drift report. Informational only; never fails.
shape:
    ./scripts/shape.sh

# Dependency vulnerability scans.
audit:
    bun audit
    go run golang.org/x/vuln/cmd/govulncheck@latest ./...

migrate:
    go run ./cmd/migrate up

migrate-down:
    go run ./cmd/migrate down

migrate-status:
    go run ./cmd/migrate status

# Start the core local stack plus optional observability tools.
# Observability UIs: GlitchTip :8000, Umami :3003, OpenObserve :5080.
infra-up:
    docker compose -p kinstack --env-file .env -f deploy/compose.yml --profile observability up -d postgres valkey mailpit rustfs gotenberg glitchtip glitchtip-postgres umami umami-postgres openobserve

# Start only the services required by `just dev`.
infra-up-core:
    docker compose -p kinstack --env-file .env -f deploy/compose.yml up -d postgres valkey mailpit rustfs gotenberg

infra-down:
    docker compose -p kinstack --env-file .env -f deploy/compose.yml --profile observability down
