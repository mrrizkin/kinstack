<script setup lang="ts">
import { router } from "@inertiajs/vue3";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import type { PageInfo } from "@/lib/contracts/pages";
import { icons } from "@/lib/icons";
import { pageSizeOptions } from "@/lib/pagination";

/**
 * The account's stored page size arrives already applied in `page.per_page`, so
 * the selector never has to know about preferences.
 */
const props = withDefaults(
  defineProps<{
    page: PageInfo;
    url: string;
    query?: Record<string, string | number>;
    /** Page-number key; the access tabs page two lists on one route. */
    pageParam?: string;
  }>(),
  { query: () => ({}), pageParam: "page" },
);

const { t } = useI18n();

const totalPages = computed(() => Math.max(props.page.total_pages, 1));
const sizeOptions = computed(() => pageSizeOptions(props.page.per_page));

function go(target: number) {
  if (target < 1 || target > totalPages.value || target === props.page.page) {
    return;
  }
  router.get(
    props.url,
    { ...props.query, [props.pageParam]: target },
    {
      preserveState: true,
      preserveScroll: true,
      replace: true,
    },
  );
}

// Dropping `page` lets the server restart at the first page of the new window.
function changeSize(value: unknown) {
  const size = Number(value);
  if (!Number.isInteger(size) || size < 1 || size === props.page.per_page) {
    return;
  }
  router.get(
    props.url,
    { ...props.query, per_page: size },
    { preserveState: true, preserveScroll: true, replace: true },
  );
}
</script>

<template>
  <div v-if="page.total > 0" class="text-muted flex flex-wrap items-center justify-between gap-3 px-1 pt-3 text-xs">
    <p>{{ t("shell.pagination.showing", { from: page.from, to: page.to, total: page.total }) }}</p>
    <div class="flex flex-wrap items-center gap-2">
      <div class="flex items-center gap-1">
        <span class="hidden sm:inline">{{ t("shell.pagination.rowsPerPage") }}</span>
        <USelect
          :model-value="String(page.per_page)"
          :items="sizeOptions"
          size="xs"
          class="w-20"
          :aria-label="t('shell.pagination.rowsPerPage')"
          @update:model-value="changeSize" />
      </div>
      <div class="flex items-center gap-1">
        <UButton
          type="button"
          size="xs"
          color="neutral"
          variant="outline"
          :icon="icons.chevronLeft"
          :label="t('shell.pagination.previous')"
          :disabled="!page.has_prev"
          @click="go(page.page - 1)" />
        <span class="px-2">{{ t("shell.pagination.page", { page: page.page, pages: totalPages }) }}</span>
        <UButton
          type="button"
          size="xs"
          color="neutral"
          variant="outline"
          :trailing-icon="icons.chevronRight"
          :label="t('shell.pagination.next')"
          :disabled="!page.has_next"
          @click="go(page.page + 1)" />
      </div>
    </div>
  </div>
</template>
