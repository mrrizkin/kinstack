<script setup lang="ts">
/**
 * Timestamp cell. Keeps the machine-readable `<time datetime>` attribute and
 * the shared date format in one place.
 */
import { formatDateTime } from "@/lib/formatters";

withDefaults(defineProps<{ value: string | null | undefined; align?: "start" | "end" }>(), {
  align: "start",
});
</script>

<template>
  <time
    :datetime="value || undefined"
    class="text-muted block text-xs whitespace-nowrap"
    :class="align === 'end' ? 'text-right' : undefined">
    {{ formatDateTime(value) }}
  </time>
</template>
