export interface Role {
  code: string;
  name: string;
  description: string;
  rank: number;
  is_system: boolean;
  permissions: string[];
}

export interface Permission {
  code: string;
  category: string;
  label: string;
  description: string;
}

export interface PermissionCategory {
  category: string;
  items: Permission[];
}

export interface Override {
  permission: string;
  label: string;
  effect: "allow" | "deny";
  reason: string;
  expires_at?: string;
  expired: boolean;
}

export interface UserAccess {
  public_id: string;
  username: string;
  display_name: string;
  role: string;
  role_name: string;
  is_active: boolean;
  overrides: Override[];
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
