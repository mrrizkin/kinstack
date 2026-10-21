setup:
    bun install
    ./scripts/setup-secrets.sh

# Regenerate empty or example credentials in .env (never rotates real ones).
setup-secrets:
    ./scripts/setup-secrets.sh

[parallel]
dev: dev-app dev-assets

dev-app:
    go tool air

dev-assets:
    bun x vite

generate-views:
    go tool templ generate

# Emails are embedded at compile time; build them when a fresh checkout lacks
# the generated directory.
ensure-email-assets:
    @ls assets/emails/dist/*.html >/dev/null 2>&1 || bun x vite build

build: build-assets build-app

build-app: ensure-email-assets generate-views
    go build -trimpath -ldflags "-X kinstack/internal/platform/inertia.buildVersion=`git rev-parse --short HEAD || echo dev`" -o ./build/kinstack ./cmd/kinstack

build-assets:
    bun x vite build

format:
    bun x oxfmt resources/
    just format-go

format-go:
    git ls-files -z --cached --others --exclude-standard -- '*.go' ':!*_templ.go' | xargs -0 -r go tool goimports -w
    git ls-files -z --cached --others --exclude-standard -- '*.go' ':!*_templ.go' | xargs -0 -r go tool gofumpt -w
    go tool golines -w --ignore-generated .

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

# Icon names must live in resources/js/lib/icons.ts, never in components.
check-icons:
    ./scripts/check-icons.sh

# Requires a production build and no running Vite dev server.
smoke-csp:
    ./scripts/smoke-csp.sh

format-check:
    bun x oxfmt --check resources/

# Go formatting gate. verify only checks; use `just format-go` to fix.
format-go-check:
    ./scripts/check-go-format.sh

test: type-check test-go test-frontend

test-go: ensure-email-assets
    go test ./...

test-frontend:
    bun test resources/js

# Print a local Go coverage report. This is informational, not a CI gate.
coverage:
    go test ./... -coverprofile=/tmp/kinstack-cover.out
    @go tool cover -func=/tmp/kinstack-cover.out | tail -n 1

# Full gate before committing: types, tests, lint, format, feature slices.
verify: type-check format-go-check test-go test-frontend lint-go lint-check format-check check-feature-slices check-icons

# Structural drift report. Informational only; never fails.
shape:
    ./scripts/shape.sh

# Dependency vulnerability scans.
# FIXME: sprintf-js has no patched release and is reachable only through
# js-yaml's CLI, which the build never runs. Drop the ignore when a fix ships.
audit:
    bun audit --ignore=GHSA-hp3w-g68c-fv3c
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
