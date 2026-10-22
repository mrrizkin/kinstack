import { z } from "zod";

import { pageInfoSchema } from "./pages";

export const roleSchema = z.looseObject({
  code: z.string(),
  name: z.string(),
  description: z.string(),
  rank: z.number().int(),
  is_system: z.boolean(),
  permissions: z.array(z.string()).default([]),
});
export type Role = z.infer<typeof roleSchema>;

export const permissionSchema = z.looseObject({
  code: z.string(),
  category: z.string(),
  label: z.string(),
  description: z.string(),
});
export type Permission = z.infer<typeof permissionSchema>;

export const overrideSchema = z.looseObject({
  permission: z.string(),
  label: z.string(),
  effect: z.enum(["allow", "deny"]),
  reason: z.string(),
  expires_at: z.string().optional(),
  expired: z.boolean(),
});
export type Override = z.infer<typeof overrideSchema>;

export const userAccessSchema = z.looseObject({
  public_id: z.string(),
  username: z.string(),
  display_name: z.string(),
  role: z.string(),
  is_active: z.boolean(),
  overrides: z.array(overrideSchema).default([]),
});
export type UserAccess = z.infer<typeof userAccessSchema>;

export const accessPageSchema = z.looseObject({
  roles: z.array(roleSchema).default([]),
  role_options: z.array(roleSchema).default([]),
  permissions: z.array(permissionSchema).default([]),
  categories: z.array(z.string()).default([]),
  users: z.array(userAccessSchema).default([]),
  page: pageInfoSchema,
  roles_page: pageInfoSchema,
});
export type AccessPageProps = z.infer<typeof accessPageSchema>;
