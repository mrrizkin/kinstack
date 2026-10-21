<script setup lang="ts">
import { usePage } from "@inertiajs/vue3";
import { computed, watch } from "vue";

import { useNotify } from "@/lib/notify";

interface FlashPayload {
  success?: string;
  error?: string;
}

const page = usePage();
const notify = useNotify();

const flash = computed(() => (page as unknown as { flash?: FlashPayload }).flash ?? undefined);

// One-shot flash: success and error become toasts.
watch(
  flash,
  (value) => {
    if (value?.success) notify("success", value.success);
    if (value?.error) notify("error", value.error);
  },
  { immediate: true, deep: true },
);
</script>

<template>
  <span class="hidden" aria-hidden="true" />
</template>
