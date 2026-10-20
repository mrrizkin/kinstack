# KinStack documentation

Module map and implementation status for the KinStack application base.

| Document | Contents |
|---|---|
| [`MODULES.md`](./MODULES.md) | Base module details and implementation status |
| [`ARCHITECTURE.md`](./ARCHITECTURE.md) | Application structure, feature slices, and platform boundaries |
| [`OBSERVABILITY.md`](./OBSERVABILITY.md) | GlitchTip, OpenObserve (OTLP), and Umami |
| [`OPERATIONS.md`](./OPERATIONS.md) | Service targets, backup/restore, deployment, rollback, and incident response |
| [`ACCESSIBILITY.md`](./ACCESSIBILITY.md) | Manual accessibility checklist for the shell, forms, and overlays |
| [`INTERNATIONALIZATION.md`](./INTERNATIONALIZATION.md) | i18n boundary, locale handling, and adding a new locale |
| [`RENOVATE.md`](./RENOVATE.md) | Renovate for Forgejo/GitHub |

## Base scope

The base provides the following generic foundation:

1. **Authentication** — username/password login (Argon2id), email magic link,
   self-service password reset, passkey/WebAuthn, optional multi-provider SSO
   (generic OIDC, Google/Apple/Facebook/GitHub via goth, SAML 2.0), break-glass
   super admin through `ACCESS_KEY`/`SECRET_KEY`, Valkey sessions, rate limits,
   CSRF, two-step verification (TOTP), and recovery codes. Personal API tokens
   (bearer) with permission scopes are available for integrations without a
   browser session. Admins can also invite new users by email (the account is
   created first, and the recipient sets a password through a one-time link).
2. **Users & authorization** — user CRUD, roles, hybrid RBAC + ABAC permissions,
   per-user permission overrides, and a fail-closed route guard.
3. **Profile & preferences** — display name, change password, theme,
   notification sound, and active session management.
4. **Audit** — append-only recording for every HTTP mutation and an audit viewer.
5. **Media** — file upload to S3-compatible object storage (RustFS) behind the
   `MEDIA_ENABLED` feature flag, with authenticated private file delivery.
6. **Notifications** — in-app notification center (list, mark read) with an
   internal service other slices can use to send notifications.
7. **Observability** — GlitchTip error tracking (Sentry-compatible), OpenObserve
   traces and logs via OTLP/HTTP, and Umami analytics. All are opt-in through
   environment variables and do not change behavior when disabled.
8. **Dashboard shell & public pages** — adapted from
   `nuxt-ui-templates/dashboard-vue` (sidebar, command palette, notification
   slide-over, light/dark theme) and `nuxt-ui-templates/saas` (landing page,
   split-screen auth layout).

Feature slices are **not** part of the base and are added as new vertical
slices.
