import type { ActiveSession, Preferences, Profile, ThemeOption, TwoFactor } from "@/lib/contracts/profile";
import type { ThemePreference } from "@/lib/theme";

export type { ActiveSession, Preferences, Profile, ThemeOption, TwoFactor };

export interface ProfileForm {
  display_name: string;
  errors: { display_name?: string };
  processing: boolean;
}

export interface PasswordForm {
  current_password: string;
  new_password: string;
  confirm_password: string;
  errors: { current_password?: string; new_password?: string; confirm_password?: string };
  processing: boolean;
}

export interface PreferencesForm {
  theme: ThemePreference;
  sound_enabled: boolean;
  page_size: number;
  errors: { theme?: string; page_size?: string };
  processing: boolean;
}

export interface SelectOption {
  label: string;
  value: string;
}

export interface TwoFactorCodeForm {
  code: string;
  errors: { code?: string };
  processing: boolean;
}

export interface TwoFactorPasswordForm {
  password: string;
  errors: { password?: string };
  processing: boolean;
}
