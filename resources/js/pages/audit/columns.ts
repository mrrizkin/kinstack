import type { TableColumn } from "@nuxt/ui/runtime/types/index.d.ts";
import { h, resolveComponent } from "vue";

import { formatDateTime } from "@/lib/formatters";

export interface Entry {
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

const ACTION_LABELS: Record<string, string> = {
  "http.read": "Page view",
  "http.mutate": "Data change",
  "background.job": "Background job",
};

export function actionLabel(action: string) {
  return ACTION_LABELS[action] ?? action;
}

export function statusColor(status: number) {
  if (status >= 500) return "error";
  if (status >= 400) return "warning";
  if (status >= 200 && status < 400) return "success";
  return "neutral";
}

export function createAuditColumns(): TableColumn<Entry>[] {
  const UBadge = resolveComponent("UBadge");

  return [
    {
      accessorKey: "occurred_at",
      header: "Time",
      cell: ({ row }) =>
        h("div", { class: "text-muted whitespace-nowrap text-xs" }, formatDateTime(row.original.occurred_at)),
    },
    {
      accessorKey: "actor_key",
      header: "User",
      cell: ({ row }) =>
        h("div", { class: "text-xs" }, [
          h("div", { class: "text-highlighted font-medium" }, row.original.actor_key || "anonymous"),
          h("div", { class: "text-muted" }, row.original.request_id || "-"),
        ]),
    },
    {
      accessorKey: "action",
      header: "Action",
      cell: ({ row }) =>
        h(UBadge, { color: "neutral", variant: "subtle", size: "sm" }, () => actionLabel(row.original.action)),
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
}
