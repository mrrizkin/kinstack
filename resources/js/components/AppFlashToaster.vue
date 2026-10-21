<script setup lang="ts">
import { usePage } from "@inertiajs/vue3";
import { computed, watch } from "vue";

import { icons } from "@/lib/icons";
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
        icon: icons.success,
        color: "success",
        duration: 4000,
      });
    }
    if (value.error) {
      playNotificationSound("error");
      toast.add({ title: value.error, icon: icons.error, color: "error", duration: 6000 });
    }
    if (value.info) {
      playNotificationSound("info");
      toast.add({ title: value.info, icon: icons.info, color: "info", duration: 5000 });
    }
  },
  { immediate: true, deep: true },
);
</script>

<template>
  <span class="hidden" aria-hidden="true" />
</template>
