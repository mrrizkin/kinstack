<script setup lang="ts">
import { router } from "@inertiajs/vue3";
import { computed } from "vue";

import { icons } from "@/lib/icons";

export interface PageInfo {
  page: number;
  per_page: number;
  total: number;
  total_pages: number;
  from: number;
  to: number;
  has_prev: boolean;
  has_next: boolean;
}

/**
 * Shared list pagination. The parent owns the filters; this component only
 * appends the target page to the current query and lets Inertia re-request.
 */
const props = withDefaults(
  defineProps<{
    page: PageInfo;
    url: string;
    query?: Record<string, unknown>;
  }>(),
  { query: () => ({}) },
);

const totalPages = computed(() => Math.max(props.page.total_pages, 1));

function go(target: number) {
  if (target < 1 || target > totalPages.value || target === props.page.page) {
    return;
  }
  router.get(props.url, { ...props.query, page: target }, { preserveState: true, preserveScroll: true, replace: true });
}
</script>

<template>
  <div v-if="page.total > 0" class="text-muted flex flex-wrap items-center justify-between gap-3 px-1 pt-3 text-xs">
    <p>Showing {{ page.from }}–{{ page.to }} of {{ page.total }}</p>
    <div class="flex items-center gap-1">
      <UButton
        type="button"
        size="xs"
        color="neutral"
        variant="outline"
        :icon="icons.chevronLeft"
        label="Previous"
        :disabled="!page.has_prev"
        @click="go(page.page - 1)" />
      <span class="px-2">Page {{ page.page }} / {{ totalPages }}</span>
      <UButton
        type="button"
        size="xs"
        color="neutral"
        variant="outline"
        :trailing-icon="icons.chevronRight"
        label="Next"
        :disabled="!page.has_next"
        @click="go(page.page + 1)" />
    </div>
  </div>
</template>
