import { z } from "zod";

export const metricsSchema = z.looseObject({
  users_total: z.number().int(),
  users_active: z.number().int(),
  roles_total: z.number().int(),
  audit_24h: z.number().int(),
  media_total: z.number().int(),
  notifications_unread: z.number().int(),
});
export type Metrics = z.infer<typeof metricsSchema>;

export const dashboardPageSchema = z.looseObject({
  metrics: metricsSchema,
});
export type DashboardPageProps = z.infer<typeof dashboardPageSchema>;
