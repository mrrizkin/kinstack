import type { ThemePreference } from "@/lib/theme";

export interface Profile {
  public_id: string;
  username: string;
  display_name: string;
  role: string;
  invited: boolean;
  last_login_at: string;
  created_at: string;
}

export interface Preferences {
  theme: string;
  sound_enabled: boolean;
  page_size: number;
}

export interface ActiveSession {
  public_id: string;
  ip: string;
  user_agent: string;
  login_at: string;
  last_seen_at: string;
  current: boolean;
}

export interface TwoFactor {
  enabled: boolean;
  pending_secret: string;
  uri: string;
  recovery_codes: string[];
}

export interface ThemeOption {
  label: string;
  value: ThemePreference;
  icon: string;
}

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
