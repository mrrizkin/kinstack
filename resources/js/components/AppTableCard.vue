<script setup lang="ts">
import AppTableTools from "@/components/AppTableTools.vue";
/**
 * Shared table wrapper, so every table keeps the same chrome and scroll.
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
