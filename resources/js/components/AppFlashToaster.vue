<script setup lang="ts">
import { usePage } from "@inertiajs/vue3";
import { computed, watch } from "vue";

import { playNotificationSound } from "@/lib/sound";

interface FlashPayload {
  success?: string;
  error?: string;
  info?: string;
}

const page = usePage();
const toast = useToast();

const flash = computed(() => (page as unknown as { flash?: FlashPayload }).flash ?? undefined);

watch(
  flash,
  (value) => {
    if (!value) return;
    if (value.success) {
      playNotificationSound("success");
      toast.add({
        title: value.success,
        icon: "i-bi-check-circle",
        color: "success",
        duration: 4000,
      });
    }
    if (value.error) {
      playNotificationSound("error");
      toast.add({ title: value.error, icon: "i-bi-x-circle", color: "error", duration: 6000 });
    }
    if (value.info) {
      playNotificationSound("info");
      toast.add({ title: value.info, icon: "i-bi-info-circle", color: "info", duration: 5000 });
    }
  },
  { immediate: true, deep: true },
);
</script>

<template>
  <span class="hidden" aria-hidden="true" />
</template>
