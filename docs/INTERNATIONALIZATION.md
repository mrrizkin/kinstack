# Internationalization

KinStack ships **English only**. The application still uses an i18n boundary so
a second locale can be added without changing response construction.

## Frontend

- `resources/js/i18n/index.ts` creates the `vue-i18n` instance with
  `locale = "en"` and `fallbackLocale = "en"`.
- `resources/js/i18n/en.ts` contains shared chrome, navigation, account menu,
  notification, and auth-layout messages.
- `app.ts` installs the i18n plugin.
- `UApp` receives the Nuxt UI English locale (`@nuxt/ui/locale`) in
  `AppLayout`, `AuthLayout`, and `PublicLayout`, so built-in labels such as
  date pickers and pagination follow English.
- Dates, relative times, numbers, and currency use `en-US` in
  `resources/js/lib/formatters.ts`. Currency remains IDR.
- Page-level copy is authored in English. When a second locale is added, move
  page copy into `resources/js/i18n/en.ts` and reference it with `t()`/`$t`.

## Go responses

- `internal/platform/i18n` is the response-message registry.
  - `T(key, args...)` translates using the default locale.
  - `TLocale(locale, key, args...)` translates explicitly.
  - `Resolve(Accept-Language)` is the request-locale hook; with one locale it
    currently resolves to English.
- Application errors, handler validation messages, flash messages,
  authorization labels/reasons, email subjects/bodies, and system help content
  use translation keys rather than inline text.
- `en.go` is the authoritative key list. Missing keys fall back to the key
  itself so gaps are visible during development.
- Config validation errors are startup/operator messages and remain outside the
  HTTP response registry.

## Email templates

Maizzle templates under `resources/js/emails` are written in English. Runtime
delimiters remain `[[.Field]]` because `{{ }}` is consumed by Vue at build time.

## Adding a locale

1. Add `resources/js/i18n/<locale>.ts` and register it in the frontend
   `messages` map.
2. Add a `catalogs[<locale>]` map in `internal/platform/i18n`.
3. Set the desired default locale in the frontend instance and Go `Default()`.
4. Add localized formatters/locale values if the new locale is not English.
5. Extend tests in `internal/platform/i18n/i18n_test.go` and
   `resources/js/i18n/i18n.test.ts`.
