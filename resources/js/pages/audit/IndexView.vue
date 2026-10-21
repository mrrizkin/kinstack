<script setup lang="ts">
import { Head, router } from "@inertiajs/vue3";
import { computed, ref } from "vue";

import AppInputDate from "@/components/AppInputDate.vue";
import AppMobileList from "@/components/AppMobileList.vue";
import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppPagination, { type PageInfo } from "@/components/AppPagination.vue";
import AppTableCard from "@/components/AppTableCard.vue";
import AppCellBadge from "@/components/cells/AppCellBadge.vue";
import AppCellStack from "@/components/cells/AppCellStack.vue";
import AppCellTime from "@/components/cells/AppCellTime.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import BaseLayout from "@/layouts/BaseLayout.vue";
import { icons } from "@/lib/icons";
import { useTableView } from "@/lib/table-view";

import { actionLabel, createAuditColumns, type Entry, outcomeColor, outcomeLabel, statusColor } from "./columns";

defineOptions({ layout: [BaseLayout, AppLayout] });

interface Filters {
  actor_key: string;
  action: string;
  outcome: string;
  search: string;
  from: string;
  to: string;
}

const props = defineProps<{
  entries: Entry[];
  filters: Filters;
  total: number;
  page: number;
  per_page: number;
  pages: number;
}>();

const ALL_FILTER = "all";

const form = ref<Filters>({
  actor_key: props.filters.actor_key ?? "",
  action: props.filters.action || ALL_FILTER,
  outcome: props.filters.outcome || ALL_FILTER,
  search: props.filters.search ?? "",
  from: props.filters.from ? props.filters.from.slice(0, 10) : "",
  to: props.filters.to ? props.filters.to.slice(0, 10) : "",
});

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

const pageInfo = computed<PageInfo>(() => ({
  page: props.page ?? 1,
  per_page: props.per_page ?? 25,
  total: props.total ?? 0,
  total_pages: props.pages ?? 0,
  from: props.total === 0 ? 0 : (props.page - 1) * props.per_page + 1,
  to: Math.min(props.page * props.per_page, props.total),
  has_prev: props.page > 1,
  has_next: props.page < props.pages,
}));

const activeFilters = computed<Record<string, string>>(() => {
  const query: Record<string, string> = {};
  if (form.value.actor_key) query.actor_key = form.value.actor_key;
  if (form.value.action !== ALL_FILTER) query.action = form.value.action;
  if (form.value.outcome !== ALL_FILTER) query.outcome = form.value.outcome;
  if (form.value.search) query.search = form.value.search;
  if (form.value.from) query.from = form.value.from;
  if (form.value.to) query.to = form.value.to;
  return query;
});

const applying = ref(false);
const hasFilters = computed(() => Object.keys(activeFilters.value).length > 0);

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

function applyFilters() {
  if (!validateFilters()) return;
  router.get("/audit", activeFilters.value, {
    preserveState: true,
    preserveScroll: true,
    replace: true,
    onStart: () => (applying.value = true),
    onFinish: () => (applying.value = false),
  });
}

function resetFilters() {
  form.value.actor_key = "";
  form.value.action = ALL_FILTER;
  form.value.outcome = ALL_FILTER;
  form.value.search = "";
  form.value.from = "";
  form.value.to = "";
  filterErrors.value.search = "";
  filterErrors.value.from = "";
  filterErrors.value.to = "";
  router.get(
    "/audit",
    {},
    {
      preserveState: true,
      preserveScroll: true,
      replace: true,
      onStart: () => (applying.value = true),
      onFinish: () => (applying.value = false),
    },
  );
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
        <UCard>
          <form class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3" @submit.prevent="applyFilters">
            <UFormField label="Search" :error="filterErrors.search || undefined">
              <UInput v-model="form.search" class="w-full" placeholder="Search activity or user details" />
            </UFormField>
            <UFormField label="User">
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
            <div class="flex items-end justify-end gap-2 sm:col-span-2 lg:col-span-3">
              <UButton
                type="button"
                color="neutral"
                variant="outline"
                label="Reset"
                :disabled="applying"
                @click="resetFilters" />
              <UButton type="submit" color="primary" :icon="icons.filter" label="Apply" :loading="applying" />
            </div>
          </form>
        </UCard>

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
          <AppPagination :page="pageInfo" url="/audit" :query="activeFilters" />
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
          <AppPagination :page="pageInfo" url="/audit" :query="activeFilters" />
        </div>
      </div>
    </template>
  </UDashboardPanel>
</template>
