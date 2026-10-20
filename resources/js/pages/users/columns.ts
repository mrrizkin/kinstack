import type { TableColumn } from "@nuxt/ui/runtime/types/index.d.ts";
import { h, resolveComponent } from "vue";

import { formatDateTime } from "@/lib/formatters";

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
}

export function createUserColumns(ctx: UserColumnsContext): TableColumn<User>[] {
  const UBadge = resolveComponent("UBadge");
  const USelect = resolveComponent("USelect");
  const UButton = resolveComponent("UButton");

  return [
    {
      accessorKey: "display_name",
      header: "User",
      cell: ({ row }) =>
        h("div", undefined, [
          h("div", { class: "text-highlighted font-medium" }, row.original.display_name),
          h("div", { class: "text-muted text-xs" }, `@${row.original.username}`),
          row.original.email ? h("div", { class: "text-muted text-xs" }, row.original.email) : null,
          row.original.invited
            ? h(UBadge, { color: "info", variant: "subtle", size: "sm", class: "mt-1" }, () => "Invited")
            : null,
        ]),
    },
    {
      accessorKey: "role",
      header: "Role",
      cell: ({ row }) =>
        h(USelect, {
          modelValue: row.original.role,
          items: ctx.roleOptions(),
          size: "xs",
          class: "w-40",
          "onUpdate:modelValue": (value: unknown) => ctx.changeRole(row.original, String(value)),
        }),
    },
    {
      accessorKey: "is_active",
      header: "Status",
      cell: ({ row }) =>
        h(
          UBadge,
          {
            color: row.original.is_active ? "success" : "error",
            variant: "subtle",
            size: "sm",
          },
          () => (row.original.is_active ? "Active" : "Inactive"),
        ),
    },
    {
      accessorKey: "last_login_at",
      header: "Last login",
      cell: ({ row }) => formatDateTime(row.original.last_login_at),
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
            variant: "outline",
            label: "Reset password",
            onClick: () => ctx.resetPassword(row.original),
          }),
          h(UButton, {
            size: "xs",
            color: row.original.is_active ? "error" : "success",
            variant: "link",
            label: row.original.is_active ? "Deactivate" : "Activate",
            onClick: () => ctx.toggleActive(row.original),
          }),
        ]),
    },
  ];
}
