# Build the frontend assets in a Bun environment.
FROM oven/bun:1.4.3 AS frontend

WORKDIR /src

COPY package.json bun.lock ./
RUN bun install --frozen-lockfile

COPY . .
RUN bun x vite build

# Build the Go application and generate server-owned view code.
FROM golang:1.27.2-bookworm AS builder

WORKDIR /src

# Asset version used by Inertia to invalidate cached pages. The build context
# excludes .git, so the revision must be supplied by the caller. When it is
# empty the build falls back to a timestamp, which keeps local images from
# sharing a stale page cache.
ARG VERSION=""

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /src/public/build ./public/build

RUN go tool templ generate
RUN CGO_ENABLED=1 GOOS=linux \
    go build -trimpath \
      -ldflags="-s -w -X kinstack/internal/platform/inertia.buildVersion=${VERSION:-$(date +%s)}" \
      -o /out/kinstack ./cmd/kinstack
RUN CGO_ENABLED=1 GOOS=linux \
    go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/cc-debian13:nonroot AS runtime

WORKDIR /app

COPY --from=builder --chown=nonroot:nonroot /out/kinstack ./kinstack
COPY --from=builder --chown=nonroot:nonroot /out/migrate ./migrate
COPY --from=builder --chown=nonroot:nonroot /src/public ./public

# Override LOG_DIR with a mounted writable path when file logs need to persist.
ENV APP_ENV=production \
    APP_PORT=3000 \
    LOG_DIR=/tmp/kinstack/logs

USER nonroot:nonroot
EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD ["/app/kinstack", "healthcheck"]

ENTRYPOINT ["/app/kinstack"]
