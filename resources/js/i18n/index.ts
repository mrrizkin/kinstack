import { createI18n } from "vue-i18n";

import { defaultLocale, fallbackLocale, messages } from "./en";

/**
 * Client-side i18n entry point.
 *
 * KinStack currently ships English only. Adding a locale means adding a catalog
 * and registering it in `messages`; no component changes are required.
 */
export const i18n = createI18n({
  legacy: false,
  locale: defaultLocale,
  fallbackLocale,
  messages: { en: messages },
});
