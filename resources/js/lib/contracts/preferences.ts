import { z } from "zod";

/**
 * Browser-owned preferences. They are parsed anyway because `localStorage` is an
 * untyped boundary.
 */
export const themePreferenceSchema = z.enum(["light", "dark", "system"]);
export type ThemePreference = z.infer<typeof themePreferenceSchema>;

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
