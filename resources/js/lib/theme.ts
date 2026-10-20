import { useColorMode } from "@vueuse/core";
import { computed } from "vue";

export type ThemePreference = "light" | "dark" | "system";

// Nuxt UI's Vue runtime shares this exact composable, so both the components and
// the toggle below stay in sync (same storage key and same <html> class).
const colorMode = useColorMode();

const STORAGE_KEY = "vueuse-color-scheme";

const toPreference = (value: string | null | undefined): ThemePreference => {
  if (value === "dark") return "dark";
  if (value === "light") return "light";
  return "system";
};

const preference = computed<ThemePreference>(() => toPreference(colorMode.store.value));
const resolved = computed<"light" | "dark">(() => (colorMode.value === "dark" ? "dark" : "light"));

/** Color mode helper shared by the layouts (light / dark / system). */
export function useTheme() {
  const setPreference = (next: ThemePreference) => {
    colorMode.store.value = next === "system" ? "auto" : next;
  };

  const cycle = () => {
    const order: ThemePreference[] = ["light", "dark", "system"];
    const current = order.indexOf(preference.value);
    const next = order[(current + 1) % order.length] ?? "system";
    setPreference(next);
  };

  return { preference, resolved, setPreference, cycle, storageKey: STORAGE_KEY };
}
