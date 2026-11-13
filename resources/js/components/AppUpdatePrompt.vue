<script setup lang="ts">
import { useRegisterSW } from "virtual:pwa-register/vue";
import { onBeforeUnmount, watch } from "vue";
import { useI18n } from "vue-i18n";

import { useNotify } from "@/lib/notify";

const UPDATE_CHECK_INTERVAL_MS = 60 * 60 * 1000;

const { t } = useI18n();
const notify = useNotify();

let timer: ReturnType<typeof setInterval> | undefined;

const { needRefresh, updateServiceWorker } = useRegisterSW({
  onRegisteredSW(_url, registration) {
    if (!registration) {
      return;
    }
    timer = setInterval(() => void registration.update(), UPDATE_CHECK_INTERVAL_MS);
  },
});

// Reloading is the only way to take a waiting worker's precache manifest.
watch(
  needRefresh,
  (available) => {
    if (!available) {
      return;
    }
    notify("info", t("shell.pwa.title"), {
      id: "pwa-update",
      description: t("shell.pwa.description"),
      duration: 0,
      close: { size: "sm" },
      silent: true,
      actions: [
        {
          label: t("shell.pwa.reload"),
          color: "primary",
          size: "xs",
          onClick: () => void updateServiceWorker(true),
        },
        {
          label: t("shell.pwa.later"),
          variant: "ghost",
          color: "neutral",
          size: "xs",
        },
      ],
    });
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  if (timer) {
    clearInterval(timer);
  }
});
</script>

<template>
  <span class="hidden" aria-hidden="true" />
</template>
