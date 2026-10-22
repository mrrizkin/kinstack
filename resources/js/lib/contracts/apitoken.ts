import { z } from "zod";

import { pageInfoSchema } from "./pages";

export const apiTokenSchema = z.looseObject({
  public_id: z.string(),
  name: z.string(),
  prefix: z.string(),
  scopes: z.array(z.string()).default([]),
  expires_at: z.string().optional(),
  last_used_at: z.string().optional(),
  created_at: z.string(),
});
export type APIToken = z.infer<typeof apiTokenSchema>;

export const apiTokensPageSchema = z.looseObject({
  tokens: z.array(apiTokenSchema).default([]),
  page: pageInfoSchema,
  created_token: z.string().optional(),
});
export type APITokensPageProps = z.infer<typeof apiTokensPageSchema>;
