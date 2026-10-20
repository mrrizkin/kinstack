# Architecture

KinStack is a modular monolith with a vertical slice per capability.

```
HTTP (Fiber)
  └── global middleware (helmet, recover, requestid, log, session, audit, csrf, inertia)
        └── bootstrap/web_routes.go   → route table
              └── internal/features/<slice>
                    handler.go   HTTP boundary: parse, validate transport, map errors
                    service.go   application logic
                    repository.go interface + SQL Result type
                    repository_pg.go PostgreSQL adapter (prepared statement)
                    repository_mem.go in-memory adapter for tests
                    dto.go       request/response payload (snake_case JSON)
                    errors.go    expected application errors
```

## Platform boundaries

| Package | Responsibility |
|---|---|
| `platform/actor` | session actor (public ID, username, role) and account validation |
| `platform/authz` | hybrid RBAC + ABAC: permission registry, roles & per-user overrides, attribute policies, route guard |
| `platform/audit` | append-only audit events for mutations |
| `platform/database` | PostgreSQL connection via pgx stdlib, pool lifetime/idle, prepared statement owner, Goose |
| `platform/email` | provider-neutral email delivery via SMTP + Maizzle template rendering (optional) |
| `platform/observability` | GlitchTip (Sentry SDK), OpenObserve OTLP traces + logs, graceful flush |
| `platform/oauth` | provider-neutral OAuth 2.0 / OIDC adapter and multi-provider registry (authorization code + PKCE, HMAC state marker, goth adapter) |
| `platform/inertia` | Fibertia adapter + document shell |
| `platform/pdf` | HTML-to-PDF renderer via Gotenberg (optional) |
| `bootstrap/fiber` | Fiber configuration (body limit, read/write/idle timeout); serves Vite build assets under the `/build` prefix from `public/build` plus public files (`/favicon.svg`) |
| `platform/password` | Argon2id hashing |
| `platform/session` | Valkey session store, pre-authentication 2FA challenge, one-time recovery codes |
| `platform/sessionregistry` | registry of active browser sessions per user (list, touch last-seen, revoke) |
| `platform/totp` | TOTP RFC 6238 (SHA-1, 6 digits, 30 seconds) |
| `platform/twofactor` | 2FA enrollment/confirmation/disable and TOTP or recovery-code verification |

## Rules

- Application logic lives only in `service.go`; handlers do not make application decisions.
- Repository interfaces live on the consumer side (feature), not in a shared tree.
- DTOs use `snake_case` JSON and must not carry `db` tags.
- SQL Result types use `db` tags and never leave the service as a response.
- Expected errors are distinguished from unexpected ones; unexpected errors are
  logged in the handler.
- Every handler has a logging path with `request_id` and `error`.
- The permission registry is owned by Go; a new slice adds its codes in
  `internal/platform/authz/catalog.go` and registers routes in `routePolicy`
  (`bootstrap/web_routes.go`). A route without a policy entry panics during
  wiring (*fail-closed*).
- `scripts/check-feature-slices.sh` verifies the canonical file shape, JSON tags,
  and the unexpected-error logger path.

## UI layer (Vue + Nuxt UI)

- Theme: `resources/css/global.css` loads Geist/Geist Mono as
  `--font-sans`/`--font-mono` and overrides the `--ui-*` variables for light and
  dark mode (palette `neutral`). `--ui-primary` is monochrome (black in light
  mode, white in dark mode) to stay consistent with the Geist template.
- `vite.config.ts` maps Nuxt UI colors (`primary: fuchsia`, `neutral:
  neutral`), Material Symbols icons (`@iconify-json/material-symbols`, classes
  `i-material-symbols-*`), and the default `subtle` variant and `sm` size for
  form controls. Bootstrap Icons (`i-bi-*`) remain available for older pages.
- Nuxt UI components are imported automatically at build time
  (`unplugin-vue-components`). Local components in `resources/js/components`
  are **not** auto-imported, so every use needs
  `import AppX from "@/components/AppX.vue"`.
- Color mode: `resources/js/lib/theme.ts` wraps `useColorMode` from
  `@vueuse/core` (key `vueuse-color-scheme`). `inertia_entry.templ` injects a
  small script in `<head>` to apply `.dark` before the first paint (no flash).
- Form controls use Nuxt UI components: `UInput`, `USelect`, `UInputDate`
  (via `AppInputDate`, value `CalendarDate` from `resources/js/lib/date.ts`),
  `UInputNumber`, `UTextarea`, `UCheckbox`, and `UFormField` for labels + error
  messages. Overlays use `UModal`/`USlideover`.
- `AppTableCard` + `UTable` is the table standard; `AppTableTools` hides columns
  and selects the row count through `useTableView`; state lasts for the browser
  session and is not yet persisted per user.
- `AppCurrencyInput` is the standard money input (`UFieldGroup` + `UInputNumber`,
  locale `id-ID`) and always returns a finite number.
- Feedback and UX clarity:
  - `AppLoadingBar` shows an Inertia progress bar during navigation/actions.
  - `AppFlashToaster` shows toasts from server flash messages
    (`platforminertia.RedirectWithFlash`).
  - Destructive actions use `AppConfirmDialog`.
- Notification sound: `resources/js/lib/sound.ts` plays a Web Audio API chime
  when a flash toast appears. The mute preference is stored in localStorage
  (`kinstack-sound`) and managed from the Preferences tab on the Profile page.
- Shared helpers: `lib/permissions.ts` (permission codes), `lib/formatters.ts`,
  `lib/date.ts`, `lib/debounce.ts`, `lib/table-view.ts`, `lib/errors.ts`.

### Dashboard shell (adapted from nuxt-ui-templates/dashboard-vue)

- `AppLayout.vue` follows the `nuxt-ui-templates/dashboard-vue` layout:
  `UDashboardGroup` + `UDashboardSidebar` (collapsible, resizable, stored in
  localStorage), `#header` `AppWorkspaceMenu`, `#default`
  `UDashboardSearchButton` + two `UNavigationMenu` instances (primary navigation
  and help), `#footer` `AppUserMenu`, plus `AppNotificationsSlideover`, which
  fetches `/notifications/feed`.
- Sidebar navigation is grouped into `Main` (Dashboard), `Administration`
  (Users, Access), and `System` (Audit, Media, Notifications). Empty groups are
  hidden after permission filtering. Account settings live in `AppUserMenu`, not
  in the sidebar.
- `UDashboardSearch` provides a command palette with the same groups. Press `N`
  to open the notification slide-over.
- `AppUserMenu` contains the user identity, profile/passkey/API token links,
  appearance submenu, help, and the sign-out action (`router.post`).
- Every page is wrapped in `UDashboardPanel` + `UDashboardNavbar` and then
  `#body` for the content.
- The base has a `dashboard` page as the landing page for every actor. A new
  slice adds its entries to the appropriate group in `AppLayout.vue` with the
  matching `permission`; if no group fits yet, add a new group and make sure its
  label only appears when at least one item passes the filter.

### Public and authentication pages

- Guests opening `/` see `pages/welcome/WelcomeView.vue` with `PublicLayout`
  (in the style of `nuxt-ui-templates/saas`: hero, feature grid, CTA). Signed-in
  actors are redirected to `/dashboard`.
- The login, forgot password, reset password, and 2FA pages use a split-screen
  `AuthLayout`: the form column on the left and a brand panel on the right, with
  a `lg`-sized `UAuthForm`. Alternative login methods (passkey, magic link,
  OAuth, SAML) remain available in the form footer according to backend feature
  flags.

## Adding a new vertical slice

1. Create `internal/features/<slice>` with the canonical files (`<slice>.go`,
   `dto.go`, `errors.go`, `handler.go`, `service.go`, `repository.go`,
   `repository_pg.go`, `repository_mem.go`, `service_test.go`).
2. Add the permission code in `internal/platform/authz/catalog.go`.
3. Register the route in `routePolicy` and mount the feature in
   `internal/bootstrap/app.go` + `web_routes.go`.
4. Add a Goose migration in `db/migrations`.
5. Add the Vue page in `resources/js/pages/<slice>/` and a navigation entry.
6. Run `just verify`.
