<script setup lang="ts">
import { onMounted } from "vue";
import { useI18n } from "vue-i18n";

import { notificationsOpen, unreadCount } from "@/lib/dashboard";

const { t } = useI18n();

async function refresh() {
  try {
    const response = await fetch("/notifications/feed", {
      headers: { Accept: "application/json" },
      credentials: "same-origin",
    });
    if (!response.ok) return;
    const payload = (await response.json()) as { unread?: number };
    unreadCount.value = payload.unread ?? 0;
  } catch {
    // The slideover shows the actionable error when opened.
  }
}

onMounted(refresh);
</script>

<template>
  <div class="relative">
    <UButton
      icon="i-material-symbols:notifications-outline-rounded"
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
