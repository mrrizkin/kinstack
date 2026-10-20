# Observability & external integrations

All integrations below are **opt-in** and do not open network connections when
disabled. The base keeps running without GlitchTip, Umami, or OpenObserve.

| Integration | Direction | Code boundary |
|---|---|---|
| GlitchTip (Sentry-compatible) | server + browser → GlitchTip | `internal/platform/observability` |
| OpenObserve (OTLP/HTTP) | server → OpenObserve | `internal/platform/observability` |
| Umami | browser → Umami | `resources/js/lib/umami.ts` |
| Renovate | Forgejo/GitHub → repo | `renovate.json`, [`RENOVATE.md`](./RENOVATE.md) |

## Local observability stack

`just infra-up` starts the core services plus the optional observability tools:

| Tool | UI |
|---|---|
| GlitchTip | <http://localhost:8000> |
| Umami | <http://localhost:3003> |
| OpenObserve | <http://localhost:5080> |

Create the resources in each tool after the stack is running. The application
integrations remain opt-in:

- **GlitchTip**: create a project in the GlitchTip UI and set
  `GLITCHTIP_DSN` to the project DSN.
- **Umami**: create a website in the Umami UI and set
  `UMAMI_ENABLED=true`, `UMAMI_SCRIPT_URL=http://localhost:3003/script.js`, and
  `UMAMI_WEBSITE_ID=<id>`.
- **OpenObserve**: set `OPENOBSERVE_ENABLED=true` and
  `OPENOBSERVE_AUTH_HEADER="Basic cm9vdEBraW5zdGFjay5sb2NhbDpLaW5TdGFjayEyMDI2"` for
  the local OpenObserve login (root@kinstack.local / KinStack!2026).

## GlitchTip

GlitchTip uses the Sentry protocol, so the SDKs in use are `sentry-go` (server)
and `@sentry/vue` (browser).

```dotenv
GLITCHTIP_DSN=https://<key>@glitchtip.example.com/<project-id>
GLITCHTIP_ENVIRONMENT=production
GLITCHTIP_RELEASE=kinstack@2026.42.0
GLITCHTIP_TRACES_SAMPLE_RATE=0.1
GLITCHTIP_DEBUG=false
```

- An empty `GLITCHTIP_DSN` disables the integration. The DSN is validated as an
  absolute URL and must use HTTPS in production.
- Server: unexpected errors (HTTP 5xx) are sent from the Fiber error handler with
  the `request_id`, `http.method`, `http.route`, `http.path`, and `trace_id`
  tags. `SendDefaultPII: false` keeps bodies and headers from being sent.
- Browser: `@sentry/vue` is initialized only when a DSN is available. Vue errors
  are captured through the existing `app.config.errorHandler` and then forwarded
  to GlitchTip.
- `GLITCHTIP_TRACES_SAMPLE_RATE` enables Sentry performance tracing (0–1). For
  OpenObserve, use the separate `OPENOBSERVE_TRACES_SAMPLE_RATE`.

## Umami

The tracker is loaded from your own Umami instance (cloud or self-hosted).

```dotenv
UMAMI_ENABLED=false
UMAMI_SCRIPT_URL=https://umami.example.com/script.js
UMAMI_WEBSITE_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
UMAMI_DOMAINS=kinstack.example.com
UMAMI_AUTO_TRACK=true
```

- The script is injected once by `installUmami()` after the Vue application is
  created.
- The Umami tracker observes the History API (`pushState`/`replaceState`), so
  Inertia navigation is recorded automatically as a page view. **Do not** call
  `umami.track()` for page views, or they will be double counted.
- Custom events use the `trackEvent(name, data)` helper from
  `resources/js/lib/umami.ts`.
- `UMAMI_DOMAINS` restricts tracking to specific domains (optional).
- The base sends no cookies and no personal data.

### CSP and nonces

CSP is enabled automatically when `APP_ENV=production` (it can be overridden with
`CSP_ENABLED`). Every document request receives a random 128-bit nonce that:

1. is written to the `Content-Security-Policy` header as `'nonce-...'`,
2. is stored in Fiber locals,
3. is passed to the Inertia renderer.

Vite uses `vite-go`'s built-in CSP support (`UseCspNonce`): the nonce is attached
to module script tags, `modulepreload`/`preload`, stylesheets, and inline prefetch
scripts. The manifest is cached by the Vite instance; per-request tag generation
is protected by a mutex because the instance stores nonce state. The inline theme
script and Inertia data block (`<script type="application/json">`) are given a
nonce by the document renderer.

External sources are derived from configuration: the Umami origin goes into
`script-src` and `connect-src`; the GlitchTip DSN origin goes into `connect-src`;
the `RUSTFS_PUBLIC_URL` origin goes into `img-src`. Manual additions:

```dotenv
CSP_ENABLED=
CSP_REPORT_ONLY=false
CSP_REPORT_URI=/csp-report
CSP_EXTRA_SCRIPT_SRC=https://cdn.example.com
CSP_EXTRA_CONNECT_SRC=https://api.example.com
CSP_EXTRA_IMG_SRC=https://images.example.com
```

Default policy: `default-src 'self'`, `object-src 'none'`, `base-uri 'self'`,
`frame-ancestors 'self'`, `style-src 'self' 'unsafe-inline'` (Nuxt UI requires
inline styles). Use `CSP_REPORT_ONLY=true` for a trial run without blocking. CSP
is intentionally disabled in development so Vite HMR keeps working.

Violation reports are sent by the browser to `CSP_REPORT_URI` (default
`/csp-report`). The `POST /csp-report` endpoint is public, limits the body to
8 KiB, and writes the report to the `csp violation report` log together with
`request_id`. This endpoint is excluded from HTTP audit so reports do not flood
the audit table.

## OpenObserve

Export uses OpenTelemetry OTLP/HTTP: **traces** and **logs** (zerolog JSON lines
are forwarded as OTLP log records).

```dotenv
OPENOBSERVE_ENABLED=false
OPENOBSERVE_ENDPOINT=http://localhost:5080
OPENOBSERVE_ORG=default
OPENOBSERVE_AUTH_HEADER=Basic <base64(email:password)>
OPENOBSERVE_STREAM=kinstack
OPENOBSERVE_SERVICE_NAME=kinstack
OPENOBSERVE_ENVIRONMENT=production
OPENOBSERVE_RELEASE=kinstack@2026.42.0
OPENOBSERVE_TRACES_SAMPLE_RATE=1
OPENOBSERVE_EXPORT_LOGS=true
OPENOBSERVE_EXPORT_METRICS=true
OPENOBSERVE_INSECURE=false
```

- OTLP endpoints are built automatically:
  - traces: `{OPENOBSERVE_ENDPOINT}/api/{OPENOBSERVE_ORG}/v1/traces`
  - logs: `{OPENOBSERVE_ENDPOINT}/api/{OPENOBSERVE_ORG}/v1/logs`
  - metrics: `{OPENOBSERVE_ENDPOINT}/api/{OPENOBSERVE_ORG}/v1/metrics`
- `OPENOBSERVE_ENDPOINT` must have **no trailing slash** (normalized on load).
- `OPENOBSERVE_AUTH_HEADER` holds the complete header value. For Basic:
  `echo -n 'email:password' | base64`, then
  `OPENOBSERVE_AUTH_HEADER=Basic <result>`. A Bearer ingestion token is also
  supported. Required in production.
- `stream-name` is sent as a header so logs land in the intended stream.
- Every HTTP request produces a server span with the `http.request.method`,
  `url.path`, `http.route`, `http.response.status_code`, and `server.address`
  attributes. `traceparent`/`tracestate` (W3C) propagation is accepted from
  clients.
- Zerolog entries (level, message, other fields) are mapped to OTel severity and
  sent through a batch processor. Logs are still written to the file as usual.
- HTTP metrics: `http.server.requests` (counter) and
  `http.server.request.duration` (histogram, seconds) with the
  `http.request.method`, `http.route`, and `http.response.status_code`
  attributes. Export runs every 60 seconds; `OPENOBSERVE_EXPORT_METRICS=false`
  disables it.
- On SIGINT/SIGTERM, the server closes connections first, then
  `Telemetry.Shutdown` flushes traces and logs (5–10 second timeout).

### Without OpenObserve

Leave `OPENOBSERVE_ENABLED=false`. No exporter, tracer provider, or additional
writer is created; application behavior stays the same as before.

## Verification

```bash
just verify
go test ./internal/platform/observability/...
```

Make sure the variables above are not filled with placeholder values in
production; `config.Load()` fails fast when the configuration is invalid.
