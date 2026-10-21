import type { TableColumn } from "@nuxt/ui/runtime/types/index.d.ts";
import { h } from "vue";

import AppCellSelect from "@/components/cells/AppCellSelect.vue";
import AppCellTime from "@/components/cells/AppCellTime.vue";
import AppStatusBadgeCell from "@/components/cells/AppStatusBadgeCell.vue";
import AppUserIdentityCell from "@/components/cells/AppUserIdentityCell.vue";

import UserActionsCell from "./components/UserActionsCell.vue";

export interface User {
  public_id: string;
  username: string;
  email?: string;
  display_name: string;
  role: string;
  is_active: boolean;
  invited: boolean;
  last_login_at?: string;
  created_at: string;
}

export interface Role {
  code: string;
  name: string;
  description: string;
  is_system: boolean;
}

export interface CreateUserForm {
  username: string;
  email: string;
  display_name: string;
  password: string;
  role: string;
  errors: {
    username?: string;
    email?: string;
    display_name?: string;
    password?: string;
    role?: string;
  };
  processing: boolean;
}

export interface InviteUserForm {
  email: string;
  username: string;
  display_name: string;
  role: string;
  errors: { email?: string; username?: string; display_name?: string; role?: string };
  processing: boolean;
}

export interface PasswordForm {
  password: string;
  errors: { password?: string };
  processing: boolean;
}

export interface UserColumnsContext {
  roleOptions: () => { label: string; value: string }[];
  changeRole(user: User, role: string): void;
  resetPassword(user: User): void;
  toggleActive(user: User): void;
  isBusy(user: User): boolean;
}

export function createUserColumns(ctx: UserColumnsContext): TableColumn<User>[] {
  return [
    {
      accessorKey: "display_name",
      header: "User",
      cell: ({ row }) =>
        h(AppUserIdentityCell, {
          name: row.original.display_name,
          username: row.original.username,
          email: row.original.email,
          badges: row.original.invited ? [{ label: "Invited", color: "info" }] : [],
        }),
    },
    {
      accessorKey: "role",
      header: "Role",
      cell: ({ row }) =>
        h(AppCellSelect, {
          modelValue: row.original.role,
          items: ctx.roleOptions(),
          busy: ctx.isBusy(row.original),
          "onUpdate:modelValue": (role: string) => ctx.changeRole(row.original, role),
        }),
    },
    {
      accessorKey: "is_active",
      header: "Status",
      cell: ({ row }) => h(AppStatusBadgeCell, { active: row.original.is_active }),
    },
    {
      accessorKey: "last_login_at",
      header: "Last login",
      cell: ({ row }) => h(AppCellTime, { value: row.original.last_login_at }),
    },
    {
      id: "actions",
      header: "Actions",
      meta: { class: { th: "text-right" } },
      cell: ({ row }) =>
        h(UserActionsCell, {
          active: row.original.is_active,
          busy: ctx.isBusy(row.original),
          onReset: () => ctx.resetPassword(row.original),
          onToggle: () => ctx.toggleActive(row.original),
        }),
    },
  ];
}
