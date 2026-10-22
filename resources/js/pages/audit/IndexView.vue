<script setup lang="ts">
import { Head, router } from "@inertiajs/vue3";
import { computed, ref, useTemplateRef } from "vue";

import AppFilterMenu from "@/components/AppFilterMenu.vue";
import AppInputDate from "@/components/AppInputDate.vue";
import AppMobileList from "@/components/AppMobileList.vue";
import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppPagination from "@/components/AppPagination.vue";
import AppQuickFilters from "@/components/AppQuickFilters.vue";
import AppTableCard from "@/components/AppTableCard.vue";
import AppCellBadge from "@/components/cells/AppCellBadge.vue";
import AppCellStack from "@/components/cells/AppCellStack.vue";
import AppCellTime from "@/components/cells/AppCellTime.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { auditPageSchema } from "@/lib/contracts/audit";
import { icons } from "@/lib/icons";
import { usePageProps, usePageSchema } from "@/lib/page-props";
import { useTableView } from "@/lib/table-view";

import { actionLabel, createAuditColumns, outcomeColor, outcomeLabel, statusColor } from "./columns";
import {
  ALL_FILTER,
  RESET_FILTER_KEY,
  createAuditPresets,
  createQuickFilterItems,
  isPresetActive,
  togglePreset,
  utcToday,
  type AuditFilterState,
} from "./presets";

defineOptions({ layout: [BaseLayout, AppLayout] });

type Filters = AuditFilterState;

const props = usePageSchema(auditPageSchema);
const { entries, filters } = props;

const { actor } = usePageProps();

/** Applied filters, normalized into the shape the form and presets share. */
const currentFilters = computed<Filters>(() => ({
  actor_key: filters.actor_key ?? "",
  action: filters.action || ALL_FILTER,
  outcome: filters.outcome || ALL_FILTER,
  search: filters.search ?? "",
  from: filters.from ? filters.from.slice(0, 10) : "",
  to: filters.to ? filters.to.slice(0, 10) : "",
}));

const form = ref<Filters>({ ...currentFilters.value });

const presets = createAuditPresets({ today: utcToday(), actorKey: actor.value?.public_id ?? "" });

const actionOptions = [
  { label: "All actions", value: ALL_FILTER },
  { label: "Page views", value: "http.read" },
  { label: "Data changes", value: "http.mutate" },
  { label: "Background jobs", value: "background.job" },
];

const outcomeOptions = [
  { label: "All outcomes", value: ALL_FILTER },
  { label: "Success", value: "success" },
  { label: "Failure", value: "failure" },
];

function toQuery(state: Filters): Record<string, string> {
  const query: Record<string, string> = {};
  if (state.actor_key) query.actor_key = state.actor_key;
  if (state.action !== ALL_FILTER) query.action = state.action;
  if (state.outcome !== ALL_FILTER) query.outcome = state.outcome;
  if (state.search) query.search = state.search;
  if (state.from) query.from = state.from;
  if (state.to) query.to = state.to;
  return query;
}

// Pagination must carry the filters the server actually applied, not a draft.
const activeFilters = computed<Record<string, string>>(() => toQuery(currentFilters.value));

const applying = ref(false);
const activeFilterCount = computed(() => Object.keys(activeFilters.value).length);
const hasFilters = computed(() => activeFilterCount.value > 0);

// Reset trails the presets so the presets keep a stable position in the row.
const quickFilters = computed(() => createQuickFilterItems(presets, currentFilters.value, icons.reset));
const filterMenu = useTemplateRef<{ close: () => void }>("filterMenu");

const MAX_SEARCH = 200;
const DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/;
const filterErrors = ref({ search: "", from: "", to: "" });

function validateFilters(): boolean {
  filterErrors.value.search =
    form.value.search.length > MAX_SEARCH ? `Search is limited to ${MAX_SEARCH} characters.` : "";
  filterErrors.value.from = form.value.from && !DATE_PATTERN.test(form.value.from) ? "Enter a valid start date." : "";
  filterErrors.value.to = form.value.to && !DATE_PATTERN.test(form.value.to) ? "Enter a valid end date." : "";
  if (
    !filterErrors.value.from &&
    !filterErrors.value.to &&
    form.value.from &&
    form.value.to &&
    form.value.to < form.value.from
  ) {
    filterErrors.value.to = "End date must be on or after the start date.";
  }
  return !filterErrors.value.search && !filterErrors.value.from && !filterErrors.value.to;
}

const emptyActions = computed(() =>
  hasFilters.value
    ? [{ label: "Reset filters", icon: icons.reset, onClick: resetFilters }]
    : [{ label: "Back to dashboard", icon: icons.arrowLeft, to: "/dashboard" }],
);

function clearFilterErrors() {
  filterErrors.value = { search: "", from: "", to: "" };
}

function navigate(query: Record<string, string>) {
  router.get("/audit", query, {
    preserveState: true,
    preserveScroll: true,
    replace: true,
    onStart: () => (applying.value = true),
    onFinish: () => (applying.value = false),
  });
}

function applyFilters() {
  if (!validateFilters()) return;
  filterMenu.value?.close();
  navigate(toQuery(form.value));
}

/**
 * A quick filter composes with the applied values, so a chip click can never
 * submit an unvalidated draft from the form. Unapplied edits are replaced.
 */
function selectQuickFilter(key: string) {
  if (key === RESET_FILTER_KEY) {
    resetFilters();
    return;
  }
  const preset = presets.find((item) => item.key === key);
  if (!preset) return;
  const next = togglePreset(currentFilters.value, preset);
  form.value = { ...next };
  clearFilterErrors();
  navigate(toQuery(next));
}

function resetFilters() {
  form.value = { actor_key: "", action: ALL_FILTER, outcome: ALL_FILTER, search: "", from: "", to: "" };
  clearFilterErrors();
  filterMenu.value?.close();
  navigate({});
}

const view = useTableView("audit", createAuditColumns(), { paged: false });
</script>

<template>
  <Head title="Audit" />
  <UDashboardPanel id="audit">
    <template #header>
      <UDashboardNavbar title="Audit">
        <template #leading>
          <UDashboardSidebarCollapse aria-label="Open or close sidebar" />
        </template>
        <span class="text-muted hidden text-sm lg:block">Activity trail and system changes.</span>
        <template #right>
          <AppNotificationBell />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="space-y-6">
        <div class="flex flex-wrap items-center justify-end gap-2">
          <AppQuickFilters class="mr-auto" :items="quickFilters" :disabled="applying" @select="selectQuickFilter" />
          <AppFilterMenu
            ref="filterMenu"
            :active-count="activeFilterCount"
            :applying="applying"
            :disabled="applying"
            @apply="applyFilters"
            @reset="resetFilters">
            <UFormField label="Search" class="col-span-2 max-md:col-span-1" :error="filterErrors.search || undefined">
              <UInput v-model="form.search" class="w-full" placeholder="Search activity or user details" />
            </UFormField>
            <UFormField label="User" class="col-span-2 max-md:col-span-1">
              <UInput v-model="form.actor_key" class="w-full" placeholder="Username or user ID" />
            </UFormField>
            <UFormField label="Action">
              <USelect v-model="form.action" :items="actionOptions" class="w-full" />
            </UFormField>
            <UFormField label="Outcome">
              <USelect v-model="form.outcome" :items="outcomeOptions" class="w-full" />
            </UFormField>
            <UFormField label="From date" :error="filterErrors.from || undefined">
              <AppInputDate v-model="form.from" placeholder="2026-01-01" class="w-full" />
            </UFormField>
            <UFormField label="To date" :error="filterErrors.to || undefined">
              <AppInputDate v-model="form.to" placeholder="2026-01-01" class="w-full" />
            </UFormField>
          </AppFilterMenu>
        </div>

        <AppTableCard :view="view" class="hidden md:block">
          <UTable :data="entries" :columns="view.columns" class="w-full">
            <template #empty>
              <UEmpty
                :icon="icons.audit"
                :title="hasFilters ? 'No matching audit entries.' : 'No audit entries yet.'"
                :description="
                  hasFilters
                    ? 'Try a different search or reset the filters.'
                    : 'Activity will appear here as users work.'
                "
                variant="naked"
                :actions="emptyActions"
                class="py-6" />
            </template>
          </UTable>
          <AppPagination :page="props.page" url="/audit" :query="activeFilters" />
        </AppTableCard>

        <div class="space-y-3 md:hidden">
          <AppMobileList v-if="entries.length > 0">
            <article v-for="entry in entries" :key="entry.id" class="space-y-3 p-4">
              <div class="flex flex-wrap items-center justify-between gap-2">
                <AppCellTime :value="entry.occurred_at" />
                <div class="flex flex-wrap items-center gap-1">
                  <AppCellBadge :color="statusColor(entry.status)" :label="String(entry.status || '-')" />
                  <AppCellBadge :color="outcomeColor(entry.outcome)" :label="outcomeLabel(entry.outcome)" />
                </div>
              </div>

              <div class="flex min-w-0 flex-wrap items-start gap-2">
                <AppCellBadge :label="actionLabel(entry.action)" />
                <AppCellStack
                  class="min-w-0 flex-1"
                  :primary="entry.resource || '-'"
                  :secondary="entry.method || '-'"
                  mono="primary"
                  truncate />
              </div>

              <div class="border-default border-t pt-2">
                <AppCellStack
                  class="min-w-0"
                  :primary="entry.actor_key || 'anonymous'"
                  :secondary="entry.request_id || '-'"
                  mono="secondary"
                  truncate />
              </div>
            </article>
          </AppMobileList>
          <UEmpty
            v-else
            :icon="icons.audit"
            :title="hasFilters ? 'No matching audit entries.' : 'No audit entries yet.'"
            :description="
              hasFilters ? 'Try a different search or reset the filters.' : 'Activity will appear here as users work.'
            "
            variant="naked"
            :actions="emptyActions"
            class="py-10" />
          <AppPagination :page="props.page" url="/audit" :query="activeFilters" />
        </div>
      </div>
    </template>
  </UDashboardPanel>
</template>
