import { z } from "zod";

import { pageInfoSchema } from "./pages";
import { themePreferenceSchema } from "./preferences";

export const profileSchema = z.looseObject({
  public_id: z.string(),
  username: z.string(),
  display_name: z.string(),
  role: z.string(),
  invited: z.boolean(),
  last_login_at: z.string(),
  created_at: z.string(),
});
export type Profile = z.infer<typeof profileSchema>;

export const preferencesSchema = z.looseObject({
  theme: z.string(),
  sound_enabled: z.boolean(),
  page_size: z.number().int(),
});
export type Preferences = z.infer<typeof preferencesSchema>;

export const activeSessionSchema = z.looseObject({
  public_id: z.string(),
  ip: z.string(),
  user_agent: z.string(),
  login_at: z.string(),
  last_seen_at: z.string(),
  current: z.boolean(),
});
export type ActiveSession = z.infer<typeof activeSessionSchema>;

export const twoFactorSchema = z.looseObject({
  enabled: z.boolean(),
  pending_secret: z.string().default(""),
  uri: z.string().default(""),
  recovery_codes: z.array(z.string()).default([]),
});
export type TwoFactor = z.infer<typeof twoFactorSchema>;

export const profilePageSchema = z.looseObject({ profile: profileSchema });
export type ProfilePageProps = z.infer<typeof profilePageSchema>;

export const preferencesPageSchema = z.looseObject({ preferences: preferencesSchema });
export type PreferencesPageProps = z.infer<typeof preferencesPageSchema>;

export const securityPageSchema = z.looseObject({
  profile: profileSchema,
  sessions: z.array(activeSessionSchema).default([]),
  sessions_page: pageInfoSchema,
  two_factor: twoFactorSchema,
});
export type SecurityPageProps = z.infer<typeof securityPageSchema>;

export const themeOptionSchema = z.object({
  label: z.string(),
  value: themePreferenceSchema,
  icon: z.string(),
});
export type ThemeOption = z.infer<typeof themeOptionSchema>;
