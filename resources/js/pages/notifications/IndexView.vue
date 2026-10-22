<script setup lang="ts">
import { Head } from "@inertiajs/vue3";
import { ref } from "vue";
import { useI18n } from "vue-i18n";
import { z } from "zod";

import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { notificationFeedSchema } from "@/lib/contracts/notifications";
import { unreadCount } from "@/lib/dashboard";
import { formatDateTime } from "@/lib/formatters";
import { requestJson } from "@/lib/http";
import { icons } from "@/lib/icons";
import { usePageProps } from "@/lib/page-props";

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

const props = defineProps<{ notifications: Notification[]; unread: number; next_cursor: string }>();

const { csrfToken } = usePageProps();
const { t } = useI18n();
const notifications = ref([...props.notifications]);
const unread = ref(props.unread);
// Keyset paging: an arrival between clicks cannot duplicate a shown row.
const nextCursor = ref(props.next_cursor);
const loadingMore = ref(false);
const error = ref("");
const markingId = ref("");
const markingAll = ref(false);

// The bell badge is shell state shared with this page, so seed it with the
// server value instead of waiting for the feed to refresh.
unreadCount.value = props.unread;

// Marking read is silent housekeeping, matching the dashboard slide-over.
async function markRead(notification: Notification) {
  if (notification.read || markingId.value) return;
  markingId.value = notification.public_id;
  error.value = "";
  try {
    await requestJson(`/notifications/${encodeURIComponent(notification.public_id)}/read`, z.unknown(), {
      method: "POST",
      csrfToken: csrfToken.value,
    });
    notification.read = true;
    unread.value = Math.max(0, unread.value - 1);
    unreadCount.value = unread.value;
  } catch {
    error.value = t("shell.notifications.markError");
  } finally {
    markingId.value = "";
  }
}

async function loadMore() {
  if (loadingMore.value || !nextCursor.value) return;
  loadingMore.value = true;
  error.value = "";
  try {
    const payload = await requestJson(
      `/notifications/feed?cursor=${encodeURIComponent(nextCursor.value)}`,
      notificationFeedSchema,
    );
    notifications.value = [...notifications.value, ...payload.notifications];
    nextCursor.value = payload.next_cursor;
    unread.value = payload.unread;
    unreadCount.value = payload.unread;
  } catch {
    error.value = t("shell.notifications.loadError");
  } finally {
    loadingMore.value = false;
  }
}

async function markAllRead() {
  if (markingAll.value || unread.value === 0) return;
  markingAll.value = true;
  error.value = "";
  try {
    await requestJson("/notifications/read-all", z.unknown(), {
      method: "POST",
      csrfToken: csrfToken.value,
    });
    notifications.value = notifications.value.map((item) => ({ ...item, read: true }));
    unread.value = 0;
    unreadCount.value = 0;
  } catch {
    error.value = t("shell.notifications.markAllError");
  } finally {
    markingAll.value = false;
  }
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
        <UAlert v-if="error" color="error" variant="subtle" :icon="icons.error" :title="error" />

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
              <p class="text-dimmed text-xs">{{ formatDateTime(notification.created_at) }}</p>
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
