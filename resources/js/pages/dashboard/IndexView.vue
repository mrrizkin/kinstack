<script setup lang="ts">
import { Head } from "@inertiajs/vue3";
import { computed } from "vue";

import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { icons } from "@/lib/icons";
import { isNavigationItemVisible } from "@/lib/navigation";
import { usePageProps } from "@/lib/page-props";
import { PERMISSIONS, usePermissions } from "@/lib/permissions";

defineOptions({ layout: [BaseLayout, AppLayout] });

interface QuickLink {
  label: string;
  description: string;
  to: string;
  icon: string;
  permission?: string;
  requiresMedia?: boolean;
}

interface Metrics {
  users_total: number;
  users_active: number;
  roles_total: number;
  audit_24h: number;
  media_total: number;
  notifications_unread: number;
}

const props = defineProps<{ metrics: Metrics }>();

const { can } = usePermissions();
const { actor, mediaEnabled } = usePageProps();

const displayName = computed(() => actor.value?.display_name || actor.value?.username || "User");
const isSuperuser = computed(() => Boolean(actor.value?.superuser));
const roleLabel = computed(() => (isSuperuser.value ? "Super Admin" : actor.value?.role || "Member"));

const now = new Date();
const greeting = computed(() => {
  const hour = now.getHours();
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  if (hour < 21) return "Good evening";
  return "Good night";
});
const formattedDate = new Intl.DateTimeFormat("en-US", {
  dateStyle: "full",
}).format(now);

const quickLinks: QuickLink[] = [
  {
    label: "Users",
    description: "Accounts, invitations, roles, and user activity status.",
    to: "/users",
    icon: icons.users,
    permission: PERMISSIONS.usersView,
  },
  {
    label: "Access",
    description: "Roles, permissions, and custom permissions for individual users.",
    to: "/access",
    icon: icons.access,
    permission: PERMISSIONS.accessManage,
  },
  {
    label: "Audit",
    description: "Activity trail and system changes.",
    to: "/audit",
    icon: icons.audit,
    permission: PERMISSIONS.auditView,
  },
  {
    label: "Media",
    description: "Files and images uploaded to storage.",
    to: "/media",
    icon: icons.media,
    permission: PERMISSIONS.mediaView,
    requiresMedia: true,
  },
  {
    label: "Notifications",
    description: "Notification center and recent activity.",
    to: "/notifications",
    icon: icons.notifications,
  },
];

const visibleQuickLinks = computed(() =>
  quickLinks.filter((link) => isNavigationItemVisible(link, can, mediaEnabled.value)),
);

interface MetricCard {
  label: string;
  value: string;
  hint: string;
  icon: string;
  to: string;
  permission?: string;
  requiresMedia?: boolean;
}

const metricCards = computed(() => {
  const cards: MetricCard[] = [
    {
      label: "Users",
      value: String(props.metrics.users_total ?? 0),
      hint: `${props.metrics.users_active ?? 0} active`,
      icon: icons.users,
      to: "/users",
      permission: PERMISSIONS.usersView,
    },
    {
      label: "Access roles",
      value: String(props.metrics.roles_total ?? 0),
      hint: "registered roles",
      icon: icons.access,
      to: "/access",
      permission: PERMISSIONS.accessManage,
    },
    {
      label: "Audit (24h)",
      value: String(props.metrics.audit_24h ?? 0),
      hint: "recent activity",
      icon: icons.audit,
      to: "/audit",
      permission: PERMISSIONS.auditView,
    },
  ];
  if (!isSuperuser.value) {
    cards.push(
      {
        label: "My media",
        value: String(props.metrics.media_total ?? 0),
        hint: "stored files",
        icon: icons.media,
        to: "/media",
        permission: PERMISSIONS.mediaView,
        requiresMedia: true,
      },
      {
        label: "Notifications",
        value: String(props.metrics.notifications_unread ?? 0),
        hint: "unread",
        icon: icons.notifications,
        to: "/notifications",
        permission: "",
      },
    );
  }
  return cards.filter((card) => isNavigationItemVisible(card, can, mediaEnabled.value));
});
</script>

<template>
  <Head title="Dashboard" />
  <UDashboardPanel id="dashboard">
    <template #header>
      <UDashboardNavbar title="Dashboard" description="Workspace overview and quick access.">
        <template #leading>
          <UDashboardSidebarCollapse aria-label="Open or close sidebar" />
        </template>
        <template #right>
          <AppNotificationBell />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="space-y-6">
        <UCard>
          <div class="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
            <div class="min-w-0">
              <p class="text-muted text-xs font-medium tracking-wide uppercase">Welcome</p>
              <h2 class="text-highlighted mt-1 truncate text-2xl font-semibold">{{ displayName }}</h2>
              <p class="text-muted mt-1 text-sm">{{ greeting }}, {{ formattedDate }}</p>
            </div>
            <div class="flex flex-wrap items-center gap-2">
              <UBadge color="neutral" variant="subtle" size="sm">{{ roleLabel }}</UBadge>
              <UBadge v-if="isSuperuser" color="warning" variant="subtle" size="sm" :icon="icons.security">
                Super admin access
              </UBadge>
              <UBadge v-else color="success" variant="subtle" size="sm" :icon="icons.success"> Active session </UBadge>
            </div>
          </div>
        </UCard>

        <section>
          <div class="mb-3">
            <h3 class="text-highlighted text-sm font-semibold">Overview</h3>
            <p class="text-muted mt-0.5 text-xs">A quick snapshot of current workspace data.</p>
          </div>
          <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <ULink v-for="card in metricCards" :key="card.label" :to="card.to" class="block">
              <UCard class="hover:bg-elevated/50 h-full transition-colors">
                <div class="flex items-start justify-between gap-3">
                  <div>
                    <p class="text-muted text-xs">{{ card.label }}</p>
                    <p class="text-highlighted mt-1 text-2xl font-semibold">{{ card.value }}</p>
                    <p class="text-dimmed mt-1 text-xs">{{ card.hint }}</p>
                  </div>
                  <UIcon :name="card.icon" class="text-dimmed size-5 shrink-0" />
                </div>
              </UCard>
            </ULink>
          </div>
        </section>

        <section>
          <div class="mb-3">
            <h3 class="text-highlighted text-sm font-semibold">Quick access</h3>
            <p class="text-muted mt-0.5 text-xs">The pages most frequently used in this workspace.</p>
          </div>
          <div class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            <ULink v-for="link in visibleQuickLinks" :key="link.to" :to="link.to" class="block">
              <UCard class="hover:bg-elevated/50 h-full transition-colors">
                <div class="flex items-start gap-3">
                  <span
                    class="bg-elevated text-toned ring-default grid size-10 shrink-0 place-items-center rounded-lg ring">
                    <UIcon :name="link.icon" class="size-5" />
                  </span>
                  <div class="min-w-0 flex-1">
                    <p class="text-highlighted font-medium">{{ link.label }}</p>
                    <p class="text-muted mt-1 text-xs leading-5">{{ link.description }}</p>
                  </div>
                  <UIcon :name="icons.external" class="text-dimmed size-4 shrink-0" />
                </div>
              </UCard>
            </ULink>
          </div>
        </section>

        <UAlert
          v-if="isSuperuser"
          color="warning"
          variant="subtle"
          :icon="icons.security"
          title="Super admin mode"
          description="This super admin account is not linked to a user profile. Profile settings, passkeys, and API tokens are unavailable; use the Users menu to manage other accounts." />
      </div>
    </template>
  </UDashboardPanel>
</template>
