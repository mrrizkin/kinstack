# Internationalization

KinStack uses English as its only locale. Both the frontend and Go responses use
translation boundaries so additional locales can be added without changing
runtime behavior.

## Frontend

- `resources/js/i18n/index.ts` creates the `vue-i18n` instance with
  `locale = "en"` and `fallbackLocale = "en"`.
- `resources/js/i18n/en.ts` contains shared chrome, navigation, account menu,
  notifications, and auth-layout messages.
- `app.ts` installs the plugin.
- `AppLayout`, `AuthLayout`, and `PublicLayout` pass the Nuxt UI English locale
  to `UApp`.
- Dates, relative times, numbers, and currency use `en-US` from
  `resources/js/lib/formatters.ts`. Currency remains IDR.
- Page copy is authored in English.

## Go responses

- `internal/platform/i18n` owns the response catalog.
- `T(key, args...)` uses the default locale.
- `TLocale(locale, key, args...)` uses an explicit locale.
- `Resolve(Accept-Language)` is the request-locale hook.
- Domain errors, handler validation, flash messages, authorization labels,
  email subjects and bodies, and help content use catalog keys.
- Missing keys fall back to the key so gaps are visible during development.
- Configuration errors are operator-facing startup messages and are not part of
  the HTTP response catalog.

## Email

Maizzle templates under `resources/js/emails` are English. Runtime placeholders
use `[[.Field]]` because `{{ }}` is consumed by Vue during the build.

## Adding a locale

1. Add `resources/js/i18n/<locale>.ts` and register it in the frontend messages.
2. Add a `catalogs[<locale>]` map in `internal/platform/i18n`.
3. Set the frontend default locale and `i18n.Default()`.
4. Add locale-specific formatters where required.
5. Extend `internal/platform/i18n/i18n_test.go` and
   `resources/js/i18n/i18n.test.ts`.
