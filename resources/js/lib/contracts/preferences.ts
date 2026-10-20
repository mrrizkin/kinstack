import { z } from "zod";

/**
 * Local, browser-owned preferences. These never come from the server, but they
 * are still parsed because `localStorage` is an untyped external boundary.
 */
export const themePreferenceSchema = z.enum(["light", "dark", "system"]);
export type ThemePreference = z.infer<typeof themePreferenceSchema>;

/** Nuxt UI / VueUse colour-mode store values. */
export const storedColorModeSchema = z.enum(["auto", "light", "dark"]);
export type StoredColorMode = z.infer<typeof storedColorModeSchema>;

export const soundPreferenceSchema = z.enum(["on", "off"]);
export type SoundPreference = z.infer<typeof soundPreferenceSchema>;

export function parseThemePreference(value: string | null | undefined): ThemePreference {
  const parsed = themePreferenceSchema.safeParse(value);
  return parsed.success ? parsed.data : "system";
}

export function parseSoundPreference(value: string | null | undefined): SoundPreference {
  const parsed = soundPreferenceSchema.safeParse(value);
  return parsed.success ? parsed.data : "on";
}
