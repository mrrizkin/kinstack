<script setup lang="ts">
import { usePage } from "@inertiajs/vue3";
import type { NavigationMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";

import AppLogo from "@/components/AppLogo.vue";
import AppUserMenu from "@/components/AppUserMenu.vue";
import { usePageProps } from "@/lib/page-props";

interface SettingsNavItem {
  label: string;
  to: string;
  icon: string;
}

interface SettingsNavGroup {
  label: string;
  items: SettingsNavItem[];
}

const page = usePage();
const { t } = useI18n();
const { authMethods } = usePageProps();
const sidebarOpen = ref(false);

const location = computed(() => {
  const url = page.url ?? "/";
  const [pathname = "/"] = url.split("?");
  return pathname.replace(/\/+$/, "") || "/";
});

const groups = computed<SettingsNavGroup[]>(() => {
  const account: SettingsNavItem[] = [
    {
      label: t("shell.settings.nav.profile"),
      to: "/profile",
      icon: "i-material-symbols:person-outline-rounded",
    },
    {
      label: t("shell.settings.nav.security"),
      to: "/profile/security",
      icon: "i-material-symbols:shield-lock-outline-rounded",
    },
    {
      label: t("shell.settings.nav.preferences"),
      to: "/profile/preferences",
      icon: "i-material-symbols:tune-rounded",
    },
  ];

  const access: SettingsNavItem[] = [];
  if (authMethods.value.passkey) {
    access.push({
      label: t("shell.settings.nav.passkeys"),
      to: "/passkeys",
      icon: "i-material-symbols:passkey-rounded",
    });
  }
  if (authMethods.value.api_token) {
    access.push({
      label: t("shell.settings.nav.apiTokens"),
      to: "/api-tokens",
      icon: "i-material-symbols:key-outline-rounded",
    });
  }

  return [
    { label: t("shell.settings.groups.account"), items: account },
    ...(access.length > 0 ? [{ label: t("shell.settings.groups.access"), items: access }] : []),
  ];
});

function isActive(item: SettingsNavItem): boolean {
  return location.value === item.to;
}

const links = computed<NavigationMenuItem[]>(() => {
  const items: NavigationMenuItem[] = [
    {
      label: t("shell.settings.back"),
      icon: "i-material-symbols:arrow-back-rounded",
      to: "/dashboard",
      onSelect: () => {
        sidebarOpen.value = false;
      },
    },
  ];
  for (const group of groups.value) {
    items.push({ type: "label", label: group.label });
    items.push(
      ...group.items.map((item) => ({
        label: item.label,
        icon: item.icon,
        to: item.to,
        active: isActive(item),
        onSelect: () => {
          sidebarOpen.value = false;
        },
      })),
    );
  }
  return items;
});
</script>

<template>
  <UDashboardGroup id="settings" unit="rem" storage="local">
    <UDashboardSidebar
      id="settings"
      v-model:open="sidebarOpen"
      collapsible
      resizable
      :default-size="16"
      :min-size="14"
      :max-size="20"
      class="bg-elevated/25"
      :ui="{ footer: 'lg:border-t lg:border-default' }">
      <template #header="{ collapsed }">
        <AppLogo :compact="collapsed" />
      </template>

      <template #default="{ collapsed }">
        <UNavigationMenu :collapsed="collapsed" :items="links" orientation="vertical" tooltip />
      </template>

      <template #footer="{ collapsed }">
        <AppUserMenu :collapsed="collapsed" />
      </template>
    </UDashboardSidebar>

    <slot />
  </UDashboardGroup>
</template>
