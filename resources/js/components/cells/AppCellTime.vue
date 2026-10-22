<script setup lang="ts">
/**
 * Timestamp cell. Keeps the machine-readable `<time datetime>` attribute and
 * the shared date format in one place.
 */
import { formatDateTime } from "@/lib/formatters";

withDefaults(defineProps<{ value: string | null | undefined; align?: "start" | "end"; inline?: boolean }>(), {
  align: "start",
  inline: false,
});
</script>

<template>
  <time
    :datetime="value || undefined"
    :class="[align === 'end' ? 'text-right' : undefined, inline ? 'whitespace-nowrap' : 'text-muted block text-xs']">
    {{ formatDateTime(value) }}
  </time>
</template>
