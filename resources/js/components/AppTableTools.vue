<script setup lang="ts">
/**
 * Per-page table controls: column visibility and rows per page.
 *
 * Rendered by AppTableCard when it receives a `view` from useTableView().
 */
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import { icons } from "@/lib/icons";
import { PAGE_SIZE_OPTIONS, type TableView } from "@/lib/table-view";

const props = defineProps<{ view: TableView }>();

const { t } = useI18n();

const perPageOptions = PAGE_SIZE_OPTIONS.map((size) => ({
  label: `${size} rows`,
  value: String(size),
}));

const perPage = computed({
  get: () => String(props.view.perPage ?? 25),
  set: (value: string) => props.view.setPerPage(Number(value)),
});
</script>

<template>
  <UPopover :ui="{ content: 'p-0' }">
    <UButton
      type="button"
      size="xs"
      color="neutral"
      variant="outline"
      :icon="icons.sliders"
      :label="t('shell.table.view')" />
    <template #content>
      <div class="w-64 space-y-3 p-3">
        <div v-if="view.paged">
          <p class="text-highlighted mb-2 text-xs font-semibold">{{ t("shell.pagination.rowsPerPage") }}</p>
          <USelect v-model="perPage" :items="perPageOptions" size="xs" class="w-full" />
        </div>
        <USeparator v-if="view.paged" />
        <div>
          <div class="mb-2 flex items-center justify-between">
            <p class="text-highlighted text-xs font-semibold">{{ t("shell.table.columns") }}</p>
            <UButton type="button" variant="link" size="xs" :label="t('shell.table.showAll')" @click="view.showAll()" />
          </div>
          <div class="max-h-56 space-y-1.5 overflow-y-auto pr-1">
            <UCheckbox
              v-for="item in view.items"
              :key="item.id"
              :model-value="item.visible"
              :label="item.label"
              @update:model-value="() => view.toggle(item.id)" />
          </div>
        </div>
      </div>
    </template>
  </UPopover>
</template>
