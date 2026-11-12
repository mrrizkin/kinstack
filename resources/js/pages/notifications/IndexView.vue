<script setup lang="ts">
import { Head } from "@inertiajs/vue3";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppCellTime from "@/components/cells/AppCellTime.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { notificationsPageSchema } from "@/lib/contracts/notifications";
import { unreadCount } from "@/lib/dashboard";
import { icons } from "@/lib/icons";
import { notificationErrorKey, useNotificationFeed } from "@/lib/notifications";
import { usePageProps, usePageSchema } from "@/lib/page-props";

defineOptions({ layout: [BaseLayout, AppLayout] });

const props = usePageSchema(notificationsPageSchema);

const { csrfToken } = usePageProps();
const { t } = useI18n();

const {
  notifications,
  unread,
  nextCursor,
  error,
  loadingMore,
  markingId,
  markingAll,
  loadMore,
  markRead,
  markAllRead,
} = useNotificationFeed({
  csrfToken: () => csrfToken.value,
  initial: { notifications: props.notifications, unread: props.unread, nextCursor: props.next_cursor },
  // The bell badge is shell state shared with this page, so the feed drives it.
  onUnread: (value) => (unreadCount.value = value),
});

const errorMessage = computed(() => (error.value ? t(notificationErrorKey(error.value)) : ""));
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
          {{ unread > 0 ? `${unread} unread` : notifications.length > 0 ? "All caught up" : "No notifications" }}
        </span>
        <template #right>
          <AppNotificationBell />
          <UButton
            v-if="unread > 0"
            color="neutral"
            variant="outline"
            :icon="icons.markAllRead"
            label="Mark all as read"
            :loading="markingAll"
            @click="markAllRead" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="space-y-3">
        <UAlert v-if="errorMessage" color="error" variant="subtle" :icon="icons.error" :title="errorMessage" />

        <UEmpty
          v-if="notifications.length === 0"
          :icon="icons.bellOff"
          title="No notifications yet."
          description="Account and workspace updates will appear here."
          variant="naked"
          :actions="[{ label: 'Back to dashboard', icon: icons.arrowLeft, to: '/dashboard' }]"
          class="py-12" />

        <UCard
          v-for="notification in notifications"
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
              <p class="text-dimmed text-xs"><AppCellTime :value="notification.created_at" inline /></p>
            </div>
            <UButton
              v-if="!notification.read"
              size="xs"
              color="neutral"
              variant="outline"
              :icon="icons.check"
              label="Mark as read"
              :loading="markingId === notification.public_id"
              @click="markRead(notification)" />
          </div>
        </UCard>

        <div v-if="nextCursor" class="flex justify-center pt-2">
          <UButton
            color="neutral"
            variant="outline"
            :icon="icons.unfoldMore"
            label="Load more"
            :loading="loadingMore"
            @click="loadMore" />
        </div>
      </div>
    </template>
  </UDashboardPanel>
</template>
