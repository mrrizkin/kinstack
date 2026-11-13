import { createI18n } from "vue-i18n";

import { defaultLocale, fallbackLocale, messages } from "./en";

/**
 * Client i18n. English only for now; a new locale adds a catalog to `messages`.
 */
export const i18n = createI18n({
  legacy: false,
  locale: defaultLocale,
  fallbackLocale,
  messages: { en: messages },
});
