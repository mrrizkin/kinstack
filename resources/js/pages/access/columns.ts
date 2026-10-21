import type { TableColumn } from "@nuxt/ui/runtime/types/index.d.ts";
import { h } from "vue";

import AppCellSelect from "@/components/cells/AppCellSelect.vue";
import AppUserIdentityCell from "@/components/cells/AppUserIdentityCell.vue";

import RoleActionsCell from "./components/RoleActionsCell.vue";
import RoleNameCell from "./components/RoleNameCell.vue";
import UserAccessActionsCell from "./components/UserAccessActionsCell.vue";
import UserOverridesCell from "./components/UserOverridesCell.vue";
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
  return [
    {
      accessorKey: "name",
      header: "Role",
      cell: ({ row }) => h(RoleNameCell, { role: row.original }),
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
      meta: { class: { th: "text-right" } },
      cell: ({ row }) =>
        h(RoleActionsCell, {
          role: row.original,
          onEdit: () => ctx.openEditRole(row.original),
          onPermissions: () => ctx.openRolePermissions(row.original),
          onRemove: () => ctx.requestDeleteRole(row.original),
        }),
    },
  ];
}

export function createUserColumns(ctx: UserColumnsContext): TableColumn<UserAccess>[] {
  return [
    {
      accessorKey: "display_name",
      header: "User",
      cell: ({ row }) =>
        h(AppUserIdentityCell, {
          name: row.original.display_name,
          username: row.original.username,
        }),
    },
    {
      accessorKey: "role",
      header: "Role",
      cell: ({ row }) =>
        h(AppCellSelect, {
          modelValue: row.original.role,
          items: ctx.roleOptions(),
          busy: ctx.assigningRoleId() === row.original.public_id,
          "onUpdate:modelValue": (role: string) => ctx.assignRole(row.original, role),
        }),
    },
    {
      id: "overrides",
      header: "Custom permissions",
      cell: ({ row }) => h(UserOverridesCell, { overrides: row.original.overrides }),
    },
    {
      id: "actions",
      header: "Actions",
      meta: { class: { th: "text-right" } },
      cell: ({ row }) => h(UserAccessActionsCell, { onManage: () => ctx.openUserOverrides(row.original) }),
    },
  ];
}
