<script setup lang="ts">
import { router } from "@inertiajs/vue3";
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { z } from "zod";

import { avatarUrl } from "@/lib/avatar";
import { notificationsOpen } from "@/lib/dashboard";
import { requestJson } from "@/lib/http";
import { icons } from "@/lib/icons";
import { usePageProps } from "@/lib/page-props";
import { useTheme, type ThemePreference } from "@/lib/theme";

const props = defineProps<{ collapsed?: boolean }>();

const { t } = useI18n();
const { preference, setPreference } = useTheme();
const { actor, csrfToken } = usePageProps();

const displayName = computed(() => actor.value?.display_name || actor.value?.username || "User");
const avatarSrc = computed(() => avatarUrl(actor.value));

const appearanceOptions = computed<{ label: string; value: ThemePreference; icon: string }[]>(() => [
  { label: t("shell.userMenu.light"), value: "light", icon: icons.light },
  { label: t("shell.userMenu.dark"), value: "dark", icon: icons.dark },
  { label: t("shell.userMenu.system"), value: "system", icon: icons.system },
]);

// The menu applies the theme instantly, then persists it for the account. The
// break-glass super admin has no profile row, so it stays browser-local.
function persistTheme(next: ThemePreference) {
  if (actor.value?.superuser) return;
  void requestJson("/profile/theme", z.unknown(), {
    method: "POST",
    body: { theme: next },
    csrfToken: csrfToken.value,
  }).catch(() => undefined);
}

const items = computed<DropdownMenuItem[][]>(() => {
  const groups: DropdownMenuItem[][] = [
    [
      {
        type: "label",
        label: displayName.value,
        avatar: { src: avatarSrc.value, alt: displayName.value, size: "sm" },
      },
    ],
  ];

  // The break-glass super admin has no profile row; /profile would redirect.
  if (!actor.value?.superuser) {
    groups.push([
      {
        label: t("shell.userMenu.settings"),
        icon: icons.settings,
        to: "/profile",
      },
    ]);
  }

  groups.push([
    {
      label: t("shell.userMenu.appearance"),
      icon: icons.palette,
      children: appearanceOptions.value.map((option) => ({
        label: option.label,
        icon: option.icon,
        type: "checkbox" as const,
        checked: preference.value === option.value,
        onSelect: (event: Event) => {
          event.preventDefault();
          setPreference(option.value);
          persistTheme(option.value);
        },
      })),
    },
  ]);

  groups.push([
    {
      label: t("shell.userMenu.notifications"),
      icon: icons.notifications,
      onSelect: () => {
        notificationsOpen.value = true;
      },
    },
    {
      label: t("shell.userMenu.help"),
      icon: icons.help,
      to: "/help",
    },
  ]);

  groups.push([
    {
      label: t("shell.userMenu.logout"),
      icon: icons.logout,
      color: "error" as const,
      onSelect: () => router.post("/logout", {}, { onSuccess: () => router.clearHistory() }),
    },
  ]);

  return groups;
});
</script>

<template>
  <UDropdownMenu
    :items="items"
    :content="{ align: 'center', collisionPadding: 12 }"
    :ui="{ content: props.collapsed ? 'w-48' : 'w-(--reka-dropdown-menu-trigger-width)' }">
    <UButton
      :avatar="{ src: avatarSrc, alt: displayName, size: 'sm' }"
      :label="props.collapsed ? undefined : displayName"
      :trailing-icon="props.collapsed ? undefined : icons.unfoldMore"
      color="neutral"
      variant="ghost"
      block
      :square="props.collapsed"
      class="data-[state=open]:bg-elevated"
      :ui="{ trailingIcon: 'text-dimmed' }" />
  </UDropdownMenu>
</template>
