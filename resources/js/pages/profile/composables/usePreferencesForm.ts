import { useForm } from "@inertiajs/vue3";
import { onMounted, watch } from "vue";

import { parseThemePreference } from "@/lib/contracts/preferences";
import { useServerErrors } from "@/lib/errors";
import { useNotificationSound } from "@/lib/sound";
import { useTheme } from "@/lib/theme";

import type { Preferences, ThemeOption } from "../types";

interface PreferencesFormProps {
  preferences: Preferences;
}

export function usePreferencesForm(props: PreferencesFormProps) {
  const { preference, setPreference, storageKey } = useTheme();
  const { enabled: soundEnabled } = useNotificationSound();

  // Start from the theme this browser is actually showing. The account value
  // below only seeds a browser that has not made a local choice yet.
  const preferencesForm = useForm({
    theme: preference.value,
    sound_enabled: props.preferences.sound_enabled,
  });

  const themeOptions: ThemeOption[] = [
    { label: "Light", value: "light", icon: "i-bi-sun" },
    { label: "Dark", value: "dark", icon: "i-bi-moon" },
    { label: "Follow system", value: "system", icon: "i-bi-display" },
  ];

  // A quick theme choice in the user menu is browser-local. Do not replace it
  // with the account preference on every settings visit: only seed a browser
  // that has never stored a choice, so another device can still pick it up.
  onMounted(() => {
    const hasLocalTheme = typeof window !== "undefined" && window.localStorage.getItem(storageKey) !== null;
    if (!hasLocalTheme) {
      const accountTheme = parseThemePreference(props.preferences.theme);
      setPreference(accountTheme);
      preferencesForm.theme = accountTheme;
    }
    if (soundEnabled.value !== props.preferences.sound_enabled) {
      soundEnabled.value = props.preferences.sound_enabled;
    }
  });

  watch(preference, (value) => {
    preferencesForm.theme = value;
  });

  function submitPreferences() {
    // Apply immediately so the change is visible before the server round-trip.
    setPreference(preferencesForm.theme);
    soundEnabled.value = preferencesForm.sound_enabled;
    preferencesForm.post("/profile/preferences", { preserveScroll: true });
  }

  const { message: fieldError, formError } = useServerErrors();

  return { preferencesForm, themeOptions, fieldError, formError, submitPreferences };
}
