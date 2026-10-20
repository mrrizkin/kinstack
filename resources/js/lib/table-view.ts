import { router } from "@inertiajs/vue3";
import type { TableColumn } from "@nuxt/ui/runtime/types/index.d.ts";
import { computed, type ComputedRef, isRef, reactive } from "vue";

/**
 * Client-side table view state (hidden columns and page size).
 *
 * The page and the table card share one module-level reactive state keyed by
 * page key, so both stay in sync. Column visibility lives only for the current
 * browser session; per-user persistence is a future general-preference feature.
 */
export interface TableViewItem {
  id: string;
  label: string;
  visible: boolean;
}

export interface TableView {
  key: string;
  columns: TableColumn<any>[];
  items: TableViewItem[];
  perPage: number | null;
  paged: boolean;
  toggle(id: string): void;
  showAll(): void;
  setPerPage(value: number | null): void;
}

export interface TableViewOptions {
  paged?: boolean;
  defaultPerPage?: number;
}

interface StoredState {
  hidden: string[];
  perPage: number | null;
}

const states = reactive<Record<string, StoredState>>({});

const PAGE_SIZE_OPTIONS = [10, 25, 50, 100];

function columnID(column: TableColumn<any>): string {
  const candidate = column as { id?: string; accessorKey?: string };
  return String(candidate.id ?? candidate.accessorKey ?? "");
}

function columnLabel(column: TableColumn<any>): string {
  const { header } = column as { header?: unknown };
  if (typeof header === "string") {
    return header;
  }
  return columnID(column);
}

function isHideable(column: TableColumn<any>): boolean {
  const id = columnID(column);
  return id !== "" && id !== "actions";
}

function ensureState(key: string): StoredState {
  const existing = states[key];
  if (existing) {
    return existing;
  }
  const created = reactive<StoredState>({ hidden: [], perPage: null });
  states[key] = created;
  return created;
}

export function useTableView(
  key: string,
  columns: TableColumn<any>[] | ComputedRef<TableColumn<any>[]>,
  options: TableViewOptions = {},
): TableView {
  const paged = options.paged ?? true;
  const defaultPerPage = options.defaultPerPage ?? 25;
  const state = ensureState(key);

  const allColumns = computed(() =>
    (isRef(columns) ? columns.value : columns).filter((column) => columnID(column) !== ""),
  );
  const visibleColumns = computed(() =>
    allColumns.value.filter((column) => !isHideable(column) || !state.hidden.includes(columnID(column))),
  );
  const items = computed<TableViewItem[]>(() =>
    allColumns.value.filter(isHideable).map((column) => ({
      id: columnID(column),
      label: columnLabel(column),
      visible: !state.hidden.includes(columnID(column)),
    })),
  );

  function toggle(id: string) {
    const index = state.hidden.indexOf(id);
    if (index >= 0) {
      state.hidden.splice(index, 1);
    } else {
      state.hidden.push(id);
    }
  }

  function showAll() {
    state.hidden = [];
  }

  function setPerPage(value: number | null) {
    state.perPage = value && value > 0 ? value : null;
    const params = new URLSearchParams(window.location.search);
    if (state.perPage) {
      params.set("per_page", String(state.perPage));
    } else {
      params.delete("per_page");
    }
    params.delete("page");
    router.get(window.location.pathname, Object.fromEntries(params.entries()), {
      preserveScroll: true,
      preserveState: true,
      replace: true,
    });
  }

  return {
    key,
    get columns() {
      return visibleColumns.value;
    },
    get items() {
      return items.value;
    },
    get perPage() {
      return state.perPage ?? defaultPerPage;
    },
    get paged() {
      return paged;
    },
    toggle,
    showAll,
    setPerPage,
  };
}

export { PAGE_SIZE_OPTIONS };
