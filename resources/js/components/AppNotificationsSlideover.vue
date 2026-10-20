<script setup lang="ts">
import { usePage } from "@inertiajs/vue3";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import { notificationsOpen, unreadCount } from "@/lib/dashboard";
import { formatRelativeTime } from "@/lib/formatters";

interface Notification {
  public_id: string;
  kind: string;
  title: string;
  body: string;
  read: boolean;
  created_at: string;
  read_at?: string;
}

const page = usePage();
const { t } = useI18n();
const csrfToken = computed(() => String(page.props.csrf_token ?? ""));

const notifications = ref<Notification[]>([]);
const unread = unreadCount;
const loading = ref(false);
const marking = ref(false);
const error = ref("");

const kindIcon = (kind: string) => {
  if (kind.includes("security") || kind.includes("auth")) {
    return "i-material-symbols:shield-outline-rounded";
  }
  if (kind.includes("media")) {
    return "i-material-symbols:image-outline-rounded";
  }
  if (kind.includes("user")) {
    return "i-material-symbols:person-outline-rounded";
  }
  return "i-material-symbols:notifications-outline-rounded";
};

async function load() {
  loading.value = true;
  error.value = "";
  try {
    const response = await fetch("/notifications/feed", {
      headers: { Accept: "application/json" },
      credentials: "same-origin",
    });
    if (!response.ok) throw new Error(t("shell.notifications.loadError"));
    const payload = (await response.json()) as { notifications?: Notification[]; unread?: number };
    notifications.value = payload.notifications ?? [];
    unread.value = payload.unread ?? 0;
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : t("shell.notifications.loadError");
  } finally {
    loading.value = false;
  }
}

async function markRead(notification: Notification) {
  if (notification.read) return;
  const response = await fetch(`/notifications/${encodeURIComponent(notification.public_id)}/read`, {
    method: "POST",
    credentials: "same-origin",
    headers: { Accept: "application/json", "X-XSRF-TOKEN": csrfToken.value },
  });
  if (!response.ok) {
    error.value = t("shell.notifications.markError");
    return;
  }
  notification.read = true;
  unread.value = Math.max(0, unread.value - 1);
}

async function markAllRead() {
  if (marking.value || unread.value === 0) return;
  marking.value = true;
  error.value = "";
  try {
    const response = await fetch("/notifications/read-all", {
      method: "POST",
      credentials: "same-origin",
      headers: { Accept: "application/json", "X-XSRF-TOKEN": csrfToken.value },
    });
    if (!response.ok) throw new Error(t("shell.notifications.markAllError"));
    notifications.value = notifications.value.map((item) => ({ ...item, read: true }));
    unread.value = 0;
  } catch (caught) {
    error.value = caught instanceof Error ? caught.message : t("shell.notifications.markAllError");
  } finally {
    marking.value = false;
  }
}

watch(notificationsOpen, (open) => {
  if (open) {
    void load();
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
            icon="i-material-symbols:done-all-rounded"
            :loading="marking"
            @click="markAllRead" />
        </div>

        <UAlert
          v-if="error"
          color="error"
          variant="subtle"
          icon="i-material-symbols:cancel-outline-rounded"
          :title="error" />

        <div v-if="loading" class="space-y-2">
          <USkeleton v-for="index in 3" :key="index" class="h-16 w-full" />
        </div>

        <div v-else-if="notifications.length === 0" class="py-12 text-center">
          <UIcon name="i-material-symbols:notifications-off-outline-rounded" class="text-dimmed mx-auto size-8" />
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
