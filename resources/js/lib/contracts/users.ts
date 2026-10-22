import { z } from "zod";

import { pageInfoSchema } from "./pages";

export const userSchema = z.looseObject({
  public_id: z.string(),
  username: z.string(),
  email: z.string().optional(),
  display_name: z.string(),
  role: z.string(),
  is_active: z.boolean(),
  invited: z.boolean(),
  last_login_at: z.string().optional(),
  created_at: z.string(),
});
export type User = z.infer<typeof userSchema>;

export const roleOptionSchema = z.looseObject({
  code: z.string(),
  name: z.string(),
  description: z.string(),
  is_system: z.boolean(),
  rank: z.number().int().optional(),
  permissions: z.array(z.string()).optional(),
});
export type RoleOption = z.infer<typeof roleOptionSchema>;

export const usersPageSchema = z.looseObject({
  users: z.array(userSchema).default([]),
  roles: z.array(roleOptionSchema).default([]),
  page: pageInfoSchema,
});
export type UsersPageProps = z.infer<typeof usersPageSchema>;
