<script setup lang="ts">
import { Head, router } from "@inertiajs/vue3";

import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { formatDateTime } from "@/lib/formatters";

defineOptions({ layout: [BaseLayout, AppLayout] });

interface Notification {
  public_id: string;
  kind: string;
  title: string;
  body: string;
  read: boolean;
  created_at: string;
  read_at?: string;
}

const props = defineProps<{ notifications: Notification[]; unread: number }>();

function markRead(notification: Notification) {
  if (notification.read) return;
  router.post(`/notifications/${notification.public_id}/read`, undefined, { preserveScroll: true });
}

function markAllRead() {
  router.post("/notifications/read-all", undefined, { preserveScroll: true });
}
</script>

<template>
  <Head title="Notifications" />
  <UDashboardPanel id="notifications">
    <template #header>
      <UDashboardNavbar title="Notifications">
        <template #leading>
          <UDashboardSidebarCollapse aria-label="Open or close sidebar" />
        </template>
        <span class="text-muted hidden text-sm lg:block">
          {{
            props.unread > 0
              ? `${props.unread} unread`
              : props.notifications.length > 0
                ? "All caught up"
                : "No notifications"
          }}
        </span>
        <template #right>
          <AppNotificationBell />
          <UButton
            v-if="props.unread > 0"
            color="neutral"
            variant="outline"
            icon="i-bi-check2-all"
            label="Mark all as read"
            @click="markAllRead" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="space-y-3">
        <div
          v-if="props.notifications.length === 0"
          class="border-default rounded-lg border border-dashed py-16 text-center">
          <UIcon name="i-bi-bell-slash" class="text-muted mx-auto size-8" />
          <p class="text-muted mt-3 text-sm">No notifications yet.</p>
        </div>

        <UCard
          v-for="notification in props.notifications"
          :key="notification.public_id"
          :class="notification.read ? '' : 'ring-primary/40 ring-1'">
          <div class="flex items-start justify-between gap-4">
            <div class="space-y-1">
              <div class="flex items-center gap-2">
                <UBadge color="neutral" variant="subtle" size="sm">{{ notification.kind }}</UBadge>
                <span v-if="!notification.read" class="bg-primary size-2 rounded-full" />
                <h3 class="text-highlighted text-sm font-semibold">{{ notification.title }}</h3>
              </div>
              <p class="text-muted text-sm">{{ notification.body }}</p>
              <p class="text-dimmed text-xs">{{ formatDateTime(notification.created_at) }}</p>
            </div>
            <UButton
              v-if="!notification.read"
              size="xs"
              color="neutral"
              variant="outline"
              icon="i-bi-check2"
              label="Mark as read"
              @click="markRead(notification)" />
          </div>
        </UCard>
      </div>
    </template>
  </UDashboardPanel>
</template>
