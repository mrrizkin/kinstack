import type { TableColumn } from "@nuxt/ui/runtime/types/index.d.ts";
import { h } from "vue";

import AppCellBadge from "@/components/cells/AppCellBadge.vue";
import AppCellStack from "@/components/cells/AppCellStack.vue";
import AppCellTime from "@/components/cells/AppCellTime.vue";
import type { CellBadgeColor } from "@/components/cells/types";
import type { AuditEntry as Entry } from "@/lib/contracts/audit";

export type { Entry };

const ACTION_LABELS: Record<string, string> = {
  "http.read": "Page view",
  "http.mutate": "Data change",
  "background.job": "Background job",
};

export function actionLabel(action: string) {
  return ACTION_LABELS[action] ?? action;
}

export function statusColor(status: number): CellBadgeColor {
  if (status >= 500) return "error";
  if (status >= 400) return "warning";
  if (status >= 200 && status < 400) return "success";
  return "neutral";
}

export function outcomeLabel(outcome: string) {
  return outcome === "failure" ? "Failure" : "Success";
}

export function outcomeColor(outcome: string): CellBadgeColor {
  return outcome === "failure" ? "error" : "success";
}

export function createAuditColumns(): TableColumn<Entry>[] {
  return [
    {
      accessorKey: "occurred_at",
      header: "Time",
      cell: ({ row }) => h(AppCellTime, { value: row.original.occurred_at }),
    },
    {
      accessorKey: "actor_key",
      header: "User",
      cell: ({ row }) =>
        h(AppCellStack, {
          primary: row.original.actor_key || "anonymous",
          secondary: row.original.request_id || "-",
          mono: "secondary",
        }),
    },
    {
      accessorKey: "action",
      header: "Action",
      cell: ({ row }) => h(AppCellBadge, { label: actionLabel(row.original.action) }),
    },
    {
      accessorKey: "resource",
      header: "Resource",
      cell: ({ row }) =>
        h(AppCellStack, {
          primary: row.original.resource || "-",
          secondary: row.original.method || "-",
          mono: "primary",
        }),
    },
    {
      accessorKey: "status",
      header: "Status",
      meta: { class: { th: "text-right" } },
      cell: ({ row }) =>
        h(AppCellBadge, {
          color: statusColor(row.original.status),
          label: String(row.original.status || "-"),
          align: "end",
        }),
    },
    {
      accessorKey: "outcome",
      header: "Outcome",
      cell: ({ row }) =>
        h(AppCellBadge, {
          color: outcomeColor(row.original.outcome),
          label: outcomeLabel(row.original.outcome),
        }),
    },
  ];
}
