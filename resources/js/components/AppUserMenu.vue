<script setup lang="ts">
import { router } from "@inertiajs/vue3";
import type { DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { notificationsOpen } from "@/lib/dashboard";
import { usePageProps } from "@/lib/page-props";
import { useTheme, type ThemePreference } from "@/lib/theme";

const props = defineProps<{ collapsed?: boolean }>();

const { t } = useI18n();
const { preference, setPreference } = useTheme();
const { actor, authMethods } = usePageProps();

const isSuperuser = computed(() => Boolean(actor.value?.superuser));
const displayName = computed(() => actor.value?.display_name || actor.value?.username || "User");
const initials = computed(() =>
  displayName.value
    .split(" ")
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join(""),
);

const appearanceOptions = computed<{ label: string; value: ThemePreference; icon: string }[]>(() => [
  { label: t("shell.userMenu.light"), value: "light", icon: "i-material-symbols:light-mode-outline-rounded" },
  { label: t("shell.userMenu.dark"), value: "dark", icon: "i-material-symbols:dark-mode-outline-rounded" },
  { label: t("shell.userMenu.system"), value: "system", icon: "i-material-symbols:desktop-windows-outline-rounded" },
]);

const items = computed<DropdownMenuItem[][]>(() => {
  const groups: DropdownMenuItem[][] = [
    [
      {
        type: "label",
        label: displayName.value,
        avatar: { alt: initials.value, size: "sm" },
      },
    ],
  ];

  if (!isSuperuser.value) {
    groups.push([
      {
        label: t("shell.userMenu.profile"),
        icon: "i-material-symbols:person-outline-rounded",
        to: "/profile",
      },
      ...(authMethods.value.api_token
        ? [
            {
              label: t("shell.userMenu.apiTokens"),
              icon: "i-material-symbols:key-outline-rounded",
              to: "/api-tokens",
            },
          ]
        : []),
      ...(authMethods.value.passkey
        ? [
            {
              label: t("shell.userMenu.passkeys"),
              icon: "i-material-symbols:passkey-rounded",
              to: "/passkeys",
            },
          ]
        : []),
    ]);
  }

  groups.push([
    {
      label: t("shell.userMenu.appearance"),
      icon: "i-material-symbols:palette-outline-rounded",
      children: appearanceOptions.value.map((option) => ({
        label: option.label,
        icon: option.icon,
        type: "checkbox" as const,
        checked: preference.value === option.value,
        onSelect: (event: Event) => {
          event.preventDefault();
          setPreference(option.value);
        },
      })),
    },
  ]);

  groups.push([
    {
      label: t("shell.userMenu.notifications"),
      icon: "i-material-symbols:notifications-outline-rounded",
      onSelect: () => {
        notificationsOpen.value = true;
      },
    },
    {
      label: t("shell.userMenu.help"),
      icon: "i-material-symbols:help-outline-rounded",
      to: "/help",
    },
  ]);

  groups.push([
    {
      label: t("shell.userMenu.logout"),
      icon: "i-material-symbols:logout-rounded",
      color: "error" as const,
      onSelect: () => router.post("/logout"),
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
      :avatar="{ alt: initials, size: 'sm' }"
      :label="props.collapsed ? undefined : displayName"
      :trailing-icon="props.collapsed ? undefined : 'i-material-symbols:unfold-more-rounded'"
      color="neutral"
      variant="ghost"
      block
      :square="props.collapsed"
      class="data-[state=open]:bg-elevated"
      :ui="{ trailingIcon: 'text-dimmed' }" />
  </UDropdownMenu>
</template>
