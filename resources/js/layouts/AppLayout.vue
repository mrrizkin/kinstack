<script setup lang="ts">
import { usePage } from "@inertiajs/vue3";
import { en as uiEn } from "@nuxt/ui/locale";
import type { NavigationMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";

import AppFlashToaster from "@/components/AppFlashToaster.vue";
import AppLoadingBar from "@/components/AppLoadingBar.vue";
import AppNotificationsSlideover from "@/components/AppNotificationsSlideover.vue";
import AppUserMenu from "@/components/AppUserMenu.vue";
import AppWorkspaceMenu from "@/components/AppWorkspaceMenu.vue";
import { notificationsOpen } from "@/lib/dashboard";
import { PERMISSIONS, usePermissions } from "@/lib/permissions";

interface NavigationItem {
  label: string;
  path: string;
  icon: string;
  permission?: string;
}

interface NavigationGroup {
  label: string;
  items: NavigationItem[];
}

const page = usePage();
const { t } = useI18n();
const { can } = usePermissions();
const isSuperuser = computed(() => Boolean((page.props.actor as { superuser?: boolean } | null)?.superuser));
const authMethods = computed(() => (page.props.auth_methods ?? {}) as { passkey?: boolean; api_token?: boolean });

const sidebarOpen = ref(false);

const primaryNavigation = computed<NavigationGroup[]>(() => [
  {
    label: t("shell.groups.main"),
    items: [
      {
        label: t("shell.nav.dashboard"),
        path: "/dashboard",
        icon: "i-material-symbols:dashboard-outline-rounded",
      },
    ],
  },
  {
    label: t("shell.groups.administration"),
    items: [
      {
        label: t("shell.nav.users"),
        path: "/users",
        icon: "i-material-symbols:group-outline-rounded",
        permission: PERMISSIONS.usersView,
      },
      {
        label: t("shell.nav.access"),
        path: "/access",
        icon: "i-material-symbols:shield-lock-outline-rounded",
        permission: PERMISSIONS.accessManage,
      },
    ],
  },
  {
    label: t("shell.groups.system"),
    items: [
      {
        label: t("shell.nav.audit"),
        path: "/audit",
        icon: "i-material-symbols:history-rounded",
        permission: PERMISSIONS.auditView,
      },
      {
        label: t("shell.nav.media"),
        path: "/media",
        icon: "i-material-symbols:image-outline-rounded",
        permission: PERMISSIONS.mediaView,
      },
      {
        label: t("shell.nav.notifications"),
        path: "/notifications",
        icon: "i-material-symbols:notifications-outline-rounded",
      },
    ],
  },
]);

const currentPath = computed(() => {
  const url = page.url ?? "/";
  const [pathname = "/"] = url.split("?");
  return pathname.replace(/\/+$/, "");
});

const visibleGroups = computed(() =>
  primaryNavigation.value
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => !item.permission || can(item.permission)),
    }))
    .filter((group) => group.items.length > 0),
);

const visibleNavigation = computed(() => visibleGroups.value.flatMap((group) => group.items));

const isActive = (path: string) => {
  const target = path.replace(/\/+$/, "");
  return currentPath.value === target || currentPath.value.startsWith(`${target}/`);
};

const activePath = computed(() => {
  const matches = visibleNavigation.value
    .map((item) => item.path)
    .filter((path) => isActive(path))
    .sort((a, b) => b.length - a.length);
  return matches[0] ?? null;
});

const toMenuItem = (item: NavigationItem): NavigationMenuItem => ({
  label: item.label,
  icon: item.icon,
  to: item.path,
  active: activePath.value === item.path,
  onSelect: () => {
    sidebarOpen.value = false;
  },
});

const links = computed<NavigationMenuItem[][]>(() => {
  const main: NavigationMenuItem[] = [];
  for (const group of visibleGroups.value) {
    main.push({ type: "label", label: group.label });
    main.push(...group.items.map(toMenuItem));
  }

  return [
    main,
    [
      {
        label: t("shell.help"),
        icon: "i-material-symbols:help-outline-rounded",
        to: "/help",
        onSelect: () => {
          sidebarOpen.value = false;
        },
      },
    ],
  ];
});

interface CommandGroupItem {
  label: string;
  icon: string;
  to: string;
}

const searchGroups = computed<{ id: string; label: string; items: CommandGroupItem[] }[]>(() =>
  [
    ...visibleGroups.value.map((group) => ({
      id: group.label.toLowerCase(),
      label: group.label,
      items: group.items.map((item) => ({
        label: item.label,
        icon: item.icon,
        to: item.path,
      })),
    })),
    {
      id: "help",
      label: t("shell.help"),
      items: [
        {
          label: t("shell.help"),
          icon: "i-material-symbols:help-outline-rounded",
          to: "/help",
        },
      ],
    },
  ].filter((group) => group.items.length > 0),
);

function onKeydown(event: KeyboardEvent) {
  if (event.key.toLowerCase() !== "n" || event.metaKey || event.ctrlKey || event.altKey) return;
  const target = event.target as HTMLElement | null;
  if (target?.closest("input, textarea, select, [contenteditable='true']")) return;
  event.preventDefault();
  notificationsOpen.value = !notificationsOpen.value;
}

onMounted(() => window.addEventListener("keydown", onKeydown));
onBeforeUnmount(() => window.removeEventListener("keydown", onKeydown));
</script>

<template>
  <UApp :locale="uiEn">
    <AppLoadingBar />
    <AppFlashToaster />

    <UDashboardGroup unit="rem" storage="local">
      <UDashboardSidebar
        id="app"
        v-model:open="sidebarOpen"
        collapsible
        resizable
        class="bg-elevated/25"
        :ui="{ footer: 'lg:border-t lg:border-default' }">
        <template #header="{ collapsed }">
          <AppWorkspaceMenu :collapsed="collapsed" />
        </template>

        <template #default="{ collapsed }">
          <UDashboardSearchButton
            :collapsed="collapsed"
            :label="t('shell.searchMenu')"
            class="ring-default bg-transparent" />

          <UNavigationMenu :collapsed="collapsed" :items="links[0]" orientation="vertical" tooltip popover />

          <UNavigationMenu :collapsed="collapsed" :items="links[1]" orientation="vertical" tooltip class="mt-auto" />
        </template>

        <template #footer="{ collapsed }">
          <AppUserMenu :collapsed="collapsed" />
        </template>
      </UDashboardSidebar>

      <UDashboardSearch :groups="searchGroups" />

      <slot />

      <AppNotificationsSlideover />
    </UDashboardGroup>
  </UApp>
</template>
