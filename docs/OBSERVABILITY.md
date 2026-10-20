# Observability

GlitchTip, OpenObserve, and Umami are optional. Each integration stays disabled
until its environment variables are configured, and the base runs without any of
them.

## Local stack

`just infra-up` starts the core services plus:

| Tool | URL |
|---|---|
| GlitchTip | <http://localhost:8000> |
| Umami | <http://localhost:3003> |
| OpenObserve | <http://localhost:5080> |

Create the project or website in each UI, then set the matching variables in
`.env`:

- `GLITCHTIP_DSN` from a GlitchTip project.
- `UMAMI_ENABLED=true`, `UMAMI_SCRIPT_URL=http://localhost:3003/script.js`, and
  `UMAMI_WEBSITE_ID` from an Umami website.
- `OPENOBSERVE_ENABLED=true` and the local `OPENOBSERVE_AUTH_HEADER` documented
  in `.env.example`.

## GlitchTip

GlitchTip uses the Sentry protocol. Server errors are captured with `sentry-go`;
browser errors use `@sentry/vue`.

```dotenv
GLITCHTIP_DSN=https://<key>@glitchtip.example.com/<project-id>
GLITCHTIP_ENVIRONMENT=production
GLITCHTIP_RELEASE=kinstack@2026.42.0
GLITCHTIP_TRACES_SAMPLE_RATE=0.1
GLITCHTIP_DEBUG=false
```

The server reports HTTP 5xx responses and attaches `request_id`, HTTP method,
route, path, and trace ID. Request bodies, headers, and user identity are not
sent. The browser SDK is loaded only when a DSN is present.

An empty DSN disables the integration. Production DSNs must use HTTPS.

## Umami

```dotenv
UMAMI_ENABLED=false
UMAMI_SCRIPT_URL=https://umami.example.com/script.js
UMAMI_WEBSITE_ID=xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
UMAMI_DOMAINS=kinstack.example.com
UMAMI_AUTO_TRACK=true
```

The tracker observes Inertia navigation through the History API, so page views
are recorded without manual calls. Use `trackEvent(name, data)` from
`resources/js/lib/umami.ts` for custom events. `UMAMI_DOMAINS` is optional.

## OpenObserve

OpenObserve receives traces, logs, and metrics over OTLP/HTTP.

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

Endpoints are derived from the base URL:

```text
{OPENOBSERVE_ENDPOINT}/api/{OPENOBSERVE_ORG}/v1/traces
{OPENOBSERVE_ENDPOINT}/api/{OPENOBSERVE_ORG}/v1/logs
{OPENOBSERVE_ENDPOINT}/api/{OPENOBSERVE_ORG}/v1/metrics
```

`OPENOBSERVE_ENDPOINT` must not end with a slash. `OPENOBSERVE_AUTH_HEADER`
contains the complete `Authorization` value, either Basic or Bearer. It is
required in production.

HTTP requests emit server spans and export the `http.server.requests` counter
and `http.server.request.duration` histogram. W3C `traceparent` and `tracestate`
headers are accepted. Logs and traces are flushed during graceful shutdown.

## CSP

CSP is enabled by default in production and can be overridden with
`CSP_ENABLED`. Development leaves it disabled so Vite HMR works.

```dotenv
CSP_ENABLED=
CSP_REPORT_ONLY=false
CSP_REPORT_URI=/csp-report
CSP_EXTRA_SCRIPT_SRC=https://cdn.example.com
CSP_EXTRA_CONNECT_SRC=https://api.example.com
CSP_EXTRA_IMG_SRC=https://images.example.com
```

Scripts use per-request nonces. Styles allow inline CSS because Nuxt UI requires
it. Umami, GlitchTip, and RustFS origins are added from their configuration.

`POST /csp-report` accepts violation reports up to 8 KiB, logs them with the
request ID, and is excluded from HTTP audit. Use `CSP_REPORT_ONLY=true` to trial
a policy without blocking resources.

## Verification

```bash
just verify
just smoke-csp
```
