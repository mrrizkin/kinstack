<script setup lang="ts">
import AppTableTools from "@/components/AppTableTools.vue";
/**
 * Shared wrapper for data tables.
 *
 * Keeps every table visually identical: the same border, radius, background,
 * and horizontal scroll. Pass a `#header` slot for an optional card title and
 * a `view` (from useTableView()) to offer column and page-size controls.
 */
import type { TableView } from "@/lib/table-view";

defineProps<{ view?: TableView }>();
</script>

<template>
  <div class="border-default bg-default overflow-hidden rounded-lg border">
    <header
      v-if="$slots.header || view"
      class="border-default flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3">
      <div class="min-w-0">
        <slot name="header" />
      </div>
      <AppTableTools v-if="view" :view="view" />
    </header>
    <div class="overflow-x-auto">
      <slot />
    </div>
  </div>
</template>
