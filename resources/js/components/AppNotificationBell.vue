<script setup lang="ts">
import { onBeforeUnmount, onMounted } from "vue";
import { useI18n } from "vue-i18n";

import { notificationFeedSchema } from "@/lib/contracts/notifications";
import { notificationsOpen, unreadCount } from "@/lib/dashboard";
import { requestJson } from "@/lib/http";
import { icons } from "@/lib/icons";
import { useNotify } from "@/lib/notify";

// Bounds feed requests while keeping the badge reasonably fresh.
const POLL_INTERVAL_MS = 45_000;

const { t } = useI18n();
const notify = useNotify();

let timer: ReturnType<typeof setInterval> | undefined;
// `undefined` means "no baseline yet": the first fetch never chimes.
let lastUnread: number | undefined;

async function refresh() {
  try {
    const payload = await requestJson("/notifications/feed", notificationFeedSchema);
    const next = payload.unread;
    if (lastUnread !== undefined && next > lastUnread) {
      const added = next - lastUnread;
      notify(
        "info",
        added === 1 ? t("shell.notifications.newOne") : t("shell.notifications.newMany", { count: added }),
      );
    }
    lastUnread = next;
    unreadCount.value = next;
  } catch {
    // Keep the last known count; the slide-over reports the actionable error.
  }
}

function stopPolling() {
  if (timer === undefined) return;
  clearInterval(timer);
  timer = undefined;
}

function startPolling() {
  stopPolling();
  timer = setInterval(() => void refresh(), POLL_INTERVAL_MS);
}

function onVisibilityChange() {
  if (document.hidden) {
    stopPolling();
    return;
  }
  // Drop the baseline so background activity never fires a stale chime.
  lastUnread = undefined;
  void refresh();
  startPolling();
}

onMounted(() => {
  void refresh();
  if (!document.hidden) startPolling();
  document.addEventListener("visibilitychange", onVisibilityChange);
});

onBeforeUnmount(() => {
  stopPolling();
  document.removeEventListener("visibilitychange", onVisibilityChange);
});
</script>

<template>
  <div class="relative">
    <UButton
      :icon="icons.notifications"
      color="neutral"
      variant="ghost"
      square
      :aria-label="t('shell.notifications.open')"
      @click="notificationsOpen = true" />
    <span
      v-if="unreadCount > 0"
      class="bg-error text-inverted pointer-events-none absolute -top-0.5 -right-0.5 grid min-w-4 place-items-center rounded-full px-1 text-[10px] leading-4 font-semibold">
      {{ unreadCount > 9 ? "9+" : unreadCount }}
    </span>
  </div>
</template>
