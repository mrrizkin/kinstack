import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";

import { icons } from "@/lib/icons";
import { usePageProps } from "@/lib/page-props";

/**
 * Section labels and copy live under `shell.settings` so every string is
 * translatable. Pages reuse the section title for their document `Head`.
 */
export interface SettingsSection {
  label: string;
  title: string;
  description: string;
  to: string;
  icon: string;
}

export interface SettingsSectionGroup {
  label: string;
  items: SettingsSection[];
}

// Module singleton keeps the settings shell free of extra providers.
export const settingsNavOpen = ref(false);

export function normalizeSettingsPath(url: string | undefined): string {
  const [pathname = "/"] = (url ?? "/").split("?");
  return pathname.replace(/\/+$/, "") || "/";
}

export function useSettingsNav() {
  const { t } = useI18n();
  const { authMethods } = usePageProps();

  const groups = computed<SettingsSectionGroup[]>(() => {
    const account: SettingsSection[] = [
      {
        label: t("shell.settings.nav.profile"),
        title: t("shell.settings.sections.profile.title"),
        description: t("shell.settings.sections.profile.description"),
        to: "/profile",
        icon: icons.account,
      },
      {
        label: t("shell.settings.nav.security"),
        title: t("shell.settings.sections.security.title"),
        description: t("shell.settings.sections.security.description"),
        to: "/profile/security",
        icon: icons.security,
      },
      {
        label: t("shell.settings.nav.preferences"),
        title: t("shell.settings.sections.preferences.title"),
        description: t("shell.settings.sections.preferences.description"),
        to: "/profile/preferences",
        icon: icons.preferences,
      },
    ];

    const access: SettingsSection[] = [];
    if (authMethods.value.passkey) {
      access.push({
        label: t("shell.settings.nav.passkeys"),
        title: t("shell.settings.sections.passkeys.title"),
        description: t("shell.settings.sections.passkeys.description"),
        to: "/passkeys",
        icon: icons.passkey,
      });
    }
    if (authMethods.value.api_token) {
      access.push({
        label: t("shell.settings.nav.apiTokens"),
        title: t("shell.settings.sections.apiTokens.title"),
        description: t("shell.settings.sections.apiTokens.description"),
        to: "/api-tokens",
        icon: icons.token,
      });
    }

    return [
      { label: t("shell.settings.groups.account"), items: account },
      ...(access.length > 0 ? [{ label: t("shell.settings.groups.access"), items: access }] : []),
    ];
  });

  const items = computed(() => groups.value.flatMap((group) => group.items));

  return { groups, items };
}
