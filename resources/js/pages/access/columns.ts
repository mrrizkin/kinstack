import type { TableColumn, DropdownMenuItem } from "@nuxt/ui/runtime/types/index.d.ts";
import { h } from "vue";

import AppCellRowMenu from "@/components/cells/AppCellRowMenu.vue";
import AppUserIdentityCell from "@/components/cells/AppUserIdentityCell.vue";

import RoleNameCell from "./components/RoleNameCell.vue";
import UserOverridesCell from "./components/UserOverridesCell.vue";
import type { Role, UserAccess } from "./types";

export interface RoleColumnsContext {
  menuItems(role: Role): DropdownMenuItem[];
}

export interface UserColumnsContext {
  roleLabel(role: string): string;
  menuItems(user: UserAccess): DropdownMenuItem[];
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
        h(AppCellRowMenu, {
          items: ctx.menuItems(row.original),
          label: `Actions for role ${row.original.name}`,
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
      cell: ({ row }) => ctx.roleLabel(row.original.role),
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
      cell: ({ row }) =>
        h(AppCellRowMenu, {
          items: ctx.menuItems(row.original),
          label: `Actions for ${row.original.display_name}`,
        }),
    },
  ];
}
