import { z } from "zod";

import { pageInfoSchema } from "./pages";

/** One in-app notification as returned by `/notifications/feed`. */
export const notificationSchema = z.looseObject({
  public_id: z.string(),
  kind: z.string(),
  title: z.string(),
  body: z.string(),
  read: z.boolean(),
  created_at: z.string(),
  read_at: z.string().optional(),
});
export type NotificationItem = z.infer<typeof notificationSchema>;

/**
 * `next_cursor` is empty on the last chunk, which hides the "Load more" control;
 * the dashboard summary ignores it.
 */
export const notificationFeedSchema = z.looseObject({
  notifications: z.array(notificationSchema).default([]),
  unread: z.number().int().nonnegative().default(0),
  next_cursor: z.string().default(""),
});

export const notificationsPageSchema = z.looseObject({
  notifications: z.array(notificationSchema).default([]),
  unread: z.number().int().nonnegative().default(0),
  next_cursor: z.string().default(""),
  page: pageInfoSchema.optional(),
});
export type NotificationsPageProps = z.infer<typeof notificationsPageSchema>;
