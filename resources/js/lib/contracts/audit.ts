import { z } from "zod";

import { pageInfoSchema } from "./pages";

export const auditEntrySchema = z.looseObject({
  id: z.number().int(),
  occurred_at: z.string(),
  request_id: z.string(),
  actor_key: z.string(),
  action: z.string(),
  resource: z.string(),
  method: z.string(),
  status: z.number().int(),
  outcome: z.string(),
  metadata: z.record(z.string(), z.string()).optional(),
});
export type AuditEntry = z.infer<typeof auditEntrySchema>;

export const auditFiltersSchema = z.looseObject({
  actor_key: z.string().default(""),
  action: z.string().default(""),
  outcome: z.string().default(""),
  search: z.string().default(""),
  from: z.string().default(""),
  to: z.string().default(""),
});
export type AuditFilters = z.infer<typeof auditFiltersSchema>;

export const auditPageSchema = z.looseObject({
  entries: z.array(auditEntrySchema).default([]),
  filters: auditFiltersSchema,
  page: pageInfoSchema,
});
export type AuditPageProps = z.infer<typeof auditPageSchema>;
