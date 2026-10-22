import type { Override, Permission, Role, UserAccess } from "@/lib/contracts/access";

export type { Override, Permission, Role, UserAccess };

export interface PermissionCategory {
  category: string;
  items: Permission[];
}

export interface RoleCreateForm {
  code: string;
  name: string;
  description: string;
  rank: number;
  errors: { code?: string; name?: string; description?: string; rank?: string };
  processing: boolean;
}

export interface RoleEditForm {
  name: string;
  description: string;
  rank: number;
  errors: { name?: string; description?: string; rank?: string };
  processing: boolean;
}

export interface OverrideForm {
  permission: string;
  effect: "allow" | "deny";
  reason: string;
  expires: string;
}
