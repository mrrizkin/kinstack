/**
 * Permission helpers mirroring the server-side authz catalog.
 *
 * The server shares the effective permission codes for the current actor as the
 * `permissions` Inertia prop. UI gating is a convenience only: every route is
 * still guarded on the server, so hiding a menu item never grants access.
 */
import { usePage } from "@inertiajs/vue3";
import { computed, type ComputedRef } from "vue";

export interface PermissionHelpers {
  permissions: ComputedRef<string[]>;
  can: (code: string) => boolean;
  canAny: (...codes: string[]) => boolean;
}

export function usePermissions(): PermissionHelpers {
  const page = usePage();
  const permissions = computed<string[]>(() => {
    const value = page.props.authz_permissions as string[] | undefined;
    return Array.isArray(value) ? value : [];
  });

  const can = (code: string) => permissions.value.includes(code);
  const canAny = (...codes: string[]) => codes.some((code) => can(code));

  return { permissions, can, canAny };
}

/** Canonical permission codes shared with the Go catalog. */
export const PERMISSIONS = {
  usersView: "users.view",
  usersManage: "users.manage",
  accessManage: "access.manage",
  auditView: "audit.view",
  mediaView: "media.view",
  mediaManage: "media.manage",
} as const;

/** Human-readable labels for permission codes shown in account and token UI. */
export const PERMISSION_LABELS: Record<string, string> = {
  [PERMISSIONS.usersView]: "View users",
  [PERMISSIONS.usersManage]: "Manage users",
  [PERMISSIONS.accessManage]: "Manage access",
  [PERMISSIONS.auditView]: "View audit",
  [PERMISSIONS.mediaView]: "View media",
  [PERMISSIONS.mediaManage]: "Manage media",
};

export function permissionLabel(code: string): string {
  return PERMISSION_LABELS[code] ?? code;
}
