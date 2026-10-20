import type { TableColumn } from "@nuxt/ui/runtime/types/index.d.ts";
import { h, resolveComponent } from "vue";

import type { Role, UserAccess } from "./types";

export interface RoleColumnsContext {
  openEditRole(role: Role): void;
  openRolePermissions(role: Role): void;
  requestDeleteRole(role: Role): void;
}

export interface UserColumnsContext {
  roleOptions: () => { label: string; value: string }[];
  assignRole(user: UserAccess, role: string): void;
  openUserOverrides(user: UserAccess): void;
  assigningRoleId: () => string | null;
}

export function createRoleColumns(ctx: RoleColumnsContext): TableColumn<Role>[] {
  const UBadge = resolveComponent("UBadge");
  const UButton = resolveComponent("UButton");

  return [
    {
      accessorKey: "name",
      header: "Role",
      cell: ({ row }) =>
        h("div", undefined, [
          h("div", { class: "text-highlighted flex items-center gap-2 font-medium" }, [
            row.original.name,
            row.original.is_system
              ? h(UBadge, { color: "neutral", variant: "subtle", size: "sm" }, () => "Built-in")
              : null,
          ]),
          h("div", { class: "text-muted text-xs" }, `${row.original.code} · ${row.original.description || "—"}`),
        ]),
    },
    {
      accessorKey: "rank",
      header: "Priority",
      cell: ({ row }) => String(row.original.rank),
    },
    {
      id: "permissions",
      header: "Permissions",
      cell: ({ row }) => `${row.original.permissions.length} permissions`,
    },
    {
      id: "actions",
      header: "Actions",
      meta: { class: { th: "text-right", td: "text-right" } },
      cell: ({ row }) =>
        h("div", { class: "flex justify-end gap-1" }, [
          h(UButton, {
            size: "xs",
            color: "neutral",
            variant: "ghost",
            label: "Edit",
            onClick: () => ctx.openEditRole(row.original),
          }),
          h(UButton, {
            size: "xs",
            color: "neutral",
            variant: "outline",
            label: "Manage permissions",
            onClick: () => ctx.openRolePermissions(row.original),
          }),
          !row.original.is_system
            ? h(UButton, {
                size: "xs",
                color: "error",
                variant: "link",
                label: "Delete",
                onClick: () => ctx.requestDeleteRole(row.original),
              })
            : null,
        ]),
    },
  ];
}

export function createUserColumns(ctx: UserColumnsContext): TableColumn<UserAccess>[] {
  const UBadge = resolveComponent("UBadge");
  const UButton = resolveComponent("UButton");
  const USelect = resolveComponent("USelect");
  const UIcon = resolveComponent("UIcon");

  return [
    {
      accessorKey: "display_name",
      header: "User",
      cell: ({ row }) =>
        h("div", undefined, [
          h("div", { class: "text-highlighted font-medium" }, row.original.display_name),
          h("div", { class: "text-muted text-xs" }, `@${row.original.username}`),
        ]),
    },
    {
      accessorKey: "role",
      header: "Role",
      cell: ({ row }) => {
        const busy = ctx.assigningRoleId() === row.original.public_id;
        return h("div", { class: "flex items-center gap-2" }, [
          h(USelect, {
            modelValue: row.original.role,
            items: ctx.roleOptions(),
            size: "xs",
            class: "w-40",
            disabled: busy,
            "onUpdate:modelValue": (value: unknown) => ctx.assignRole(row.original, String(value)),
          }),
          busy ? h(UIcon, { name: "i-bi-arrow-repeat", class: "text-muted size-3.5 animate-spin" }) : null,
        ]);
      },
    },
    {
      id: "overrides",
      header: "Custom permissions",
      cell: ({ row }) =>
        row.original.overrides.length === 0
          ? h("span", { class: "text-muted text-xs" }, "None")
          : h(
              "div",
              { class: "flex flex-wrap gap-1" },
              row.original.overrides.map((override) =>
                h(
                  UBadge,
                  {
                    color: override.expired ? "neutral" : override.effect === "allow" ? "success" : "error",
                    variant: "subtle",
                    size: "sm",
                  },
                  () => `${override.effect === "allow" ? "+" : "−"} ${override.label}`,
                ),
              ),
            ),
    },
    {
      id: "actions",
      header: "Actions",
      meta: { class: { th: "text-right", td: "text-right" } },
      cell: ({ row }) =>
        h(UButton, {
          size: "xs",
          color: "neutral",
          variant: "outline",
          label: "Manage permissions",
          onClick: () => ctx.openUserOverrides(row.original),
        }),
    },
  ];
}
