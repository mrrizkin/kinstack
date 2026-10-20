<script setup lang="ts">
import { Head, router } from "@inertiajs/vue3";
import type { TableColumn } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed, h, reactive, resolveComponent } from "vue";

import AppInputDate from "@/components/AppInputDate.vue";
import AppMobileList from "@/components/AppMobileList.vue";
import AppNotificationBell from "@/components/AppNotificationBell.vue";
import AppPagination, { type PageInfo } from "@/components/AppPagination.vue";
import AppTableCard from "@/components/AppTableCard.vue";
import AppLayout from "@/layouts/AppLayout.vue";
import { formatDateTime } from "@/lib/formatters";
import { useTableView } from "@/lib/table-view";

interface Entry {
  id: number;
  occurred_at: string;
  request_id: string;
  actor_key: string;
  action: string;
  resource: string;
  method: string;
  status: number;
  outcome: string;
  metadata?: Record<string, string>;
}

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

const form = reactive<Filters>({
  actor_key: props.filters.actor_key ?? "",
  action: props.filters.action || ALL_FILTER,
  outcome: props.filters.outcome || ALL_FILTER,
  search: props.filters.search ?? "",
  from: props.filters.from ? props.filters.from.slice(0, 10) : "",
  to: props.filters.to ? props.filters.to.slice(0, 10) : "",
});

const actionOptions = [
  { label: "All actions", value: ALL_FILTER },
  { label: "Read", value: "http.read" },
  { label: "Mutation", value: "http.mutate" },
  { label: "Background job", value: "background.job" },
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
  if (form.actor_key) query.actor_key = form.actor_key;
  if (form.action !== ALL_FILTER) query.action = form.action;
  if (form.outcome !== ALL_FILTER) query.outcome = form.outcome;
  if (form.search) query.search = form.search;
  if (form.from) query.from = form.from;
  if (form.to) query.to = form.to;
  return query;
});

function applyFilters() {
  router.get("/audit", activeFilters.value, { preserveState: true, preserveScroll: true, replace: true });
}

function resetFilters() {
  form.actor_key = "";
  form.action = ALL_FILTER;
  form.outcome = ALL_FILTER;
  form.search = "";
  form.from = "";
  form.to = "";
  router.get("/audit", {}, { preserveState: true, preserveScroll: true, replace: true });
}

const UBadge = resolveComponent("UBadge");

function statusColor(status: number) {
  if (status >= 500) return "error";
  if (status >= 400) return "warning";
  if (status >= 200 && status < 400) return "success";
  return "neutral";
}

const columns: TableColumn<Entry>[] = [
  {
    accessorKey: "occurred_at",
    header: "Time",
    cell: ({ row }) =>
      h("div", { class: "text-muted whitespace-nowrap text-xs" }, formatDateTime(row.original.occurred_at)),
  },
  {
    accessorKey: "actor_key",
    header: "Actor",
    cell: ({ row }) =>
      h("div", { class: "text-xs" }, [
        h("div", { class: "text-highlighted font-medium" }, row.original.actor_key || "anonymous"),
        h("div", { class: "text-muted" }, row.original.request_id || "-"),
      ]),
  },
  {
    accessorKey: "action",
    header: "Action",
    cell: ({ row }) => h(UBadge, { color: "neutral", variant: "subtle", size: "sm" }, () => row.original.action),
  },
  {
    accessorKey: "resource",
    header: "Resource",
    cell: ({ row }) =>
      h("div", { class: "text-xs" }, [
        h("div", { class: "text-highlighted font-mono" }, row.original.resource || "-"),
        h("div", { class: "text-muted" }, row.original.method || "-"),
      ]),
  },
  {
    accessorKey: "status",
    header: "Status",
    meta: { class: { th: "text-right", td: "text-right" } },
    cell: ({ row }) =>
      h("div", { class: "flex justify-end" }, [
        h(UBadge, { color: statusColor(row.original.status), variant: "subtle", size: "sm" }, () =>
          String(row.original.status || "-"),
        ),
      ]),
  },
  {
    accessorKey: "outcome",
    header: "Outcome",
    cell: ({ row }) =>
      h(
        UBadge,
        {
          color: row.original.outcome === "failure" ? "error" : "success",
          variant: "subtle",
          size: "sm",
        },
        () => (row.original.outcome === "failure" ? "Failure" : "Success"),
      ),
  },
];

const view = useTableView("audit", columns, { paged: false });
</script>

<template>
  <Head title="Audit" />
  <AppLayout>
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
              <UFormField label="Search">
                <UInput v-model="form.search" class="w-full" placeholder="resource, request id, actor" />
              </UFormField>
              <UFormField label="Actor">
                <UInput v-model="form.actor_key" class="w-full" placeholder="actor public ID" />
              </UFormField>
              <UFormField label="Action">
                <USelect v-model="form.action" :items="actionOptions" class="w-full" />
              </UFormField>
              <UFormField label="Outcome">
                <USelect v-model="form.outcome" :items="outcomeOptions" class="w-full" />
              </UFormField>
              <UFormField label="From date">
                <AppInputDate v-model="form.from" placeholder="2026-01-01" class="w-full" />
              </UFormField>
              <UFormField label="To date">
                <AppInputDate v-model="form.to" placeholder="2026-01-01" class="w-full" />
              </UFormField>
              <div class="flex items-end justify-end gap-2 sm:col-span-2 lg:col-span-3">
                <UButton type="button" color="neutral" variant="outline" label="Reset" @click="resetFilters" />
                <UButton type="submit" color="primary" icon="i-bi-funnel" label="Apply" />
              </div>
            </form>
          </UCard>

          <AppTableCard :view="view" class="hidden md:block">
            <UTable :data="entries" :columns="view.columns" class="w-full">
              <template #empty>
                <UEmpty icon="i-bi-journal-text" title="No audit entries yet." variant="naked" class="py-6" />
              </template>
            </UTable>
            <AppPagination :page="pageInfo" url="/audit" :query="activeFilters" />
          </AppTableCard>

          <div class="space-y-3 md:hidden">
            <AppMobileList v-if="entries.length > 0">
              <article v-for="entry in entries" :key="entry.id" class="space-y-3 p-4">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <time :datetime="entry.occurred_at" class="text-muted text-xs">
                    {{ formatDateTime(entry.occurred_at) }}
                  </time>
                  <div class="flex flex-wrap items-center gap-1">
                    <UBadge :color="statusColor(entry.status)" variant="subtle" size="sm">
                      {{ entry.status || "-" }}
                    </UBadge>
                    <UBadge :color="entry.outcome === 'failure' ? 'error' : 'success'" variant="subtle" size="sm">
                      {{ entry.outcome === "failure" ? "Failure" : "Success" }}
                    </UBadge>
                  </div>
                </div>

                <div class="min-w-0">
                  <div class="flex flex-wrap items-center gap-2">
                    <UBadge color="neutral" variant="subtle" size="sm">{{ entry.action }}</UBadge>
                    <span class="text-highlighted truncate font-mono text-xs">{{ entry.resource || "-" }}</span>
                  </div>
                  <p class="text-muted mt-1 text-xs">{{ entry.method || "-" }}</p>
                </div>

                <div class="border-default space-y-1 border-t pt-2">
                  <p class="text-toned truncate text-xs font-medium">{{ entry.actor_key || "anonymous" }}</p>
                  <p class="text-muted truncate font-mono text-[11px]">{{ entry.request_id || "-" }}</p>
                </div>
              </article>
            </AppMobileList>
            <UEmpty v-else icon="i-bi-journal-text" title="No audit entries yet." variant="naked" class="py-10" />
            <AppPagination :page="pageInfo" url="/audit" :query="activeFilters" />
          </div>
        </div>
      </template>
    </UDashboardPanel>
  </AppLayout>
</template>
