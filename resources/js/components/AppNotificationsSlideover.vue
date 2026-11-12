<script setup lang="ts">
import { computed, watch } from "vue";
import { useI18n } from "vue-i18n";

import { notificationsOpen, unreadCount } from "@/lib/dashboard";
import { formatRelativeTime } from "@/lib/formatters";
import { icons } from "@/lib/icons";
import { notificationErrorKey, useNotificationFeed } from "@/lib/notifications";
import { usePageProps } from "@/lib/page-props";

const { t } = useI18n();
const { csrfToken } = usePageProps();

const { notifications, unread, error, loading, markingAll, refresh, markRead, markAllRead } = useNotificationFeed({
  csrfToken: () => csrfToken.value,
  initial: { unread: unreadCount.value },
  onUnread: (value) => (unreadCount.value = value),
});

const errorMessage = computed(() => (error.value ? t(notificationErrorKey(error.value)) : ""));

const kindIcon = (kind: string) => {
  if (kind.includes("security") || kind.includes("auth")) {
    return icons.security;
  }
  if (kind.includes("media")) {
    return icons.media;
  }
  if (kind.includes("user")) {
    return icons.account;
  }
  return icons.notifications;
};

watch(notificationsOpen, (open) => {
  if (open) {
    void refresh();
  }
});
</script>

<template>
  <USlideover v-model:open="notificationsOpen" :title="t('shell.notifications.title')">
    <template #body>
      <div class="space-y-4">
        <div class="flex items-center justify-between gap-3">
          <p class="text-muted text-sm">
            {{
              unread > 0
                ? t("shell.notifications.unread", { count: unread })
                : notifications.length > 0
                  ? t("shell.notifications.allRead")
                  : t("shell.notifications.none")
            }}
          </p>
          <UButton
            v-if="unread > 0"
            :label="t('shell.notifications.markAll')"
            color="neutral"
            variant="ghost"
            size="xs"
            :icon="icons.markAllRead"
            :loading="markingAll"
            @click="markAllRead" />
        </div>

        <UAlert v-if="errorMessage" color="error" variant="subtle" :icon="icons.error" :title="errorMessage" />

        <div v-if="loading" class="space-y-2">
          <USkeleton v-for="index in 3" :key="index" class="h-16 w-full" />
        </div>

        <div v-else-if="notifications.length === 0" class="py-12 text-center">
          <UIcon :name="icons.bellOff" class="text-dimmed mx-auto size-8" />
          <p class="text-muted mt-3 text-sm">{{ t("shell.notifications.none") }}</p>
        </div>

        <div v-else class="space-y-1">
          <button
            v-for="notification in notifications"
            :key="notification.public_id"
            type="button"
            class="hover:bg-elevated/50 flex w-full items-start gap-3 rounded-md px-3 py-2.5 text-left transition-colors"
            :class="notification.read ? 'opacity-75' : ''"
            @click="markRead(notification)">
            <UChip color="error" :show="!notification.read" inset>
              <span
                class="bg-elevated text-toned ring-default grid size-9 shrink-0 place-items-center rounded-full ring">
                <UIcon :name="kindIcon(notification.kind)" class="size-4" />
              </span>
            </UChip>

            <span class="min-w-0 flex-1 text-sm">
              <span class="flex items-start justify-between gap-3">
                <span class="text-highlighted font-medium">{{ notification.title }}</span>
                <time :datetime="notification.created_at" class="text-muted shrink-0 text-xs">{{
                  formatRelativeTime(notification.created_at)
                }}</time>
              </span>
              <span class="text-dimmed mt-0.5 line-clamp-2 block">{{ notification.body }}</span>
            </span>
          </button>
        </div>

        <UButton
          :label="t('shell.notifications.openCenter')"
          color="neutral"
          variant="outline"
          block
          to="/notifications"
          @click="notificationsOpen = false" />
      </div>
    </template>
  </USlideover>
</template>
