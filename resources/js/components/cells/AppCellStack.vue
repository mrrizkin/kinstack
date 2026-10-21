<script setup lang="ts">
/**
 * Primary and secondary text lines for a table cell.
 *
 * Props only, so the cell can be rendered from `h()` in a column definition.
 * `mono` marks identifier lines (request ids, resource names, role codes).
 */
import AppCellBadge from "@/components/cells/AppCellBadge.vue";
import type { CellBadgeSpec } from "@/components/cells/types";

withDefaults(
  defineProps<{
    primary: string;
    secondary?: string;
    tertiary?: string;
    badges?: CellBadgeSpec[];
    mono?: "none" | "primary" | "secondary" | "both";
    truncate?: boolean;
    align?: "start" | "end";
  }>(),
  { badges: () => [], mono: "none", truncate: false, align: "start" },
);
</script>

<template>
  <div :class="align === 'end' ? 'text-right' : undefined">
    <div
      class="text-highlighted flex flex-wrap items-center gap-2 font-medium"
      :class="[{ 'font-mono': mono === 'primary' || mono === 'both' }, { truncate }]">
      {{ primary }}
      <AppCellBadge v-for="badge in badges" :key="badge.label" :label="badge.label" :color="badge.color" />
    </div>
    <div
      v-if="secondary"
      class="text-muted text-xs"
      :class="[{ 'font-mono': mono === 'secondary' || mono === 'both' }, { truncate }]">
      {{ secondary }}
    </div>
    <div v-if="tertiary" class="text-muted text-xs" :class="{ truncate }">{{ tertiary }}</div>
  </div>
</template>
