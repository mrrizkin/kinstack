import { computed, ref, shallowRef, type ComputedRef, type Ref } from "vue";
import { z } from "zod";

import { notificationFeedSchema, type NotificationItem } from "./contracts/notifications";
import { requestJson, type RequestJsonOptions } from "./http";

/**
 * Notification feed lifecycle, shared by the page and the dashboard slide-over.
 *
 * | from | event | to |
 * | --- | --- | --- |
 * | idle | refresh | loading |
 * | idle | loadMore | loadingMore |
 * | idle | mark | marking |
 * | idle | markAll | markingAll |
 * | loading / loadingMore / marking / markingAll | settled | idle |
 * | any | failed | idle |
 *
 * `idle` is the only state that accepts work; `busy` is derived from the stage.
 */
type FeedStage =
  | { status: "idle" }
  | { status: "loading" }
  | { status: "loadingMore" }
  | { status: "marking"; id: string }
  | { status: "markingAll" };

type FeedEvent =
  | { type: "refresh" }
  | { type: "loadMore" }
  | { type: "mark"; id: string }
  | { type: "markAll" }
  | { type: "settled" };

function transition(stage: FeedStage, event: FeedEvent): FeedStage {
  if (event.type === "settled") {
    return { status: "idle" };
  }
  if (stage.status !== "idle") {
    throw new Error(`notifications: ${event.type} is not allowed from ${stage.status}`);
  }
  switch (event.type) {
    case "refresh":
      return { status: "loading" };
    case "loadMore":
      return { status: "loadingMore" };
    case "markAll":
      return { status: "markingAll" };
    case "mark":
      return { status: "marking", id: event.id };
  }
}

export type NotificationError = "load" | "loadMore" | "mark" | "markAll";

/** The i18n key for a failed feed interaction. */
export function notificationErrorKey(error: NotificationError): string {
  switch (error) {
    case "load":
      return "shell.notifications.loadError";
    case "loadMore":
      return "shell.notifications.loadError";
    case "mark":
      return "shell.notifications.markError";
    case "markAll":
      return "shell.notifications.markAllError";
  }
}

type Requester = <T>(path: string, schema: z.ZodType<T>, options?: RequestJsonOptions) => Promise<T>;

export interface NotificationFeedOptions {
  csrfToken: () => string;
  initial?: { notifications?: NotificationItem[]; unread?: number; nextCursor?: string };
  /** Keeps the shell badge in sync with the feed. */
  onUnread?: (unread: number) => void;
  /** Test seam: the JSON boundary is injected, not imported at call time. */
  request?: Requester;
}

export interface NotificationFeed {
  notifications: Ref<NotificationItem[]>;
  unread: Ref<number>;
  nextCursor: Ref<string>;
  error: Ref<NotificationError | null>;
  busy: ComputedRef<boolean>;
  loading: ComputedRef<boolean>;
  loadingMore: ComputedRef<boolean>;
  markingId: ComputedRef<string | null>;
  markingAll: ComputedRef<boolean>;
  refresh(): Promise<void>;
  loadMore(): Promise<void>;
  markRead(notification: NotificationItem): Promise<void>;
  markAllRead(): Promise<void>;
}

export function useNotificationFeed(options: NotificationFeedOptions): NotificationFeed {
  const request = options.request ?? requestJson;
  const stage = shallowRef<FeedStage>({ status: "idle" });
  const notifications = ref<NotificationItem[]>([...(options.initial?.notifications ?? [])]);
  const unread = ref(options.initial?.unread ?? 0);
  const nextCursor = ref(options.initial?.nextCursor ?? "");
  const error = ref<NotificationError | null>(null);

  function send(event: FeedEvent) {
    stage.value = transition(stage.value, event);
  }

  function settle(value: number) {
    unread.value = value;
    options.onUnread?.(value);
  }

  async function refresh(): Promise<void> {
    if (stage.value.status !== "idle") {
      return;
    }
    send({ type: "refresh" });
    error.value = null;
    try {
      const payload = await request("/notifications/feed?limit=10", notificationFeedSchema);
      notifications.value = [...payload.notifications];
      nextCursor.value = payload.next_cursor;
      settle(payload.unread);
    } catch {
      error.value = "load";
    } finally {
      send({ type: "settled" });
    }
  }

  async function loadMore(): Promise<void> {
    if (stage.value.status !== "idle" || nextCursor.value === "") {
      return;
    }
    send({ type: "loadMore" });
    error.value = null;
    try {
      const payload = await request(
        `/notifications/feed?cursor=${encodeURIComponent(nextCursor.value)}`,
        notificationFeedSchema,
      );
      // Keyset paging: an arrival between clicks cannot duplicate a row.
      notifications.value = [...notifications.value, ...payload.notifications];
      nextCursor.value = payload.next_cursor;
      settle(payload.unread);
    } catch {
      error.value = "loadMore";
    } finally {
      send({ type: "settled" });
    }
  }

  async function markRead(notification: NotificationItem): Promise<void> {
    if (stage.value.status !== "idle" || notification.read) {
      return;
    }
    send({ type: "mark", id: notification.public_id });
    error.value = null;
    try {
      await request(`/notifications/${encodeURIComponent(notification.public_id)}/read`, z.unknown(), {
        method: "POST",
        csrfToken: options.csrfToken(),
      });
    } catch {
      error.value = "mark";
      return;
    } finally {
      send({ type: "settled" });
    }
    notification.read = true;
    settle(Math.max(0, unread.value - 1));
  }

  async function markAllRead(): Promise<void> {
    if (stage.value.status !== "idle" || unread.value === 0) {
      return;
    }
    send({ type: "markAll" });
    error.value = null;
    try {
      await request("/notifications/read-all", z.unknown(), {
        method: "POST",
        csrfToken: options.csrfToken(),
      });
    } catch {
      error.value = "markAll";
      return;
    } finally {
      send({ type: "settled" });
    }
    notifications.value = notifications.value.map((item) => ({ ...item, read: true }));
    settle(0);
  }

  return {
    notifications,
    unread,
    nextCursor,
    error,
    busy: computed(() => stage.value.status !== "idle"),
    loading: computed(() => stage.value.status === "loading"),
    loadingMore: computed(() => stage.value.status === "loadingMore"),
    markingId: computed(() => (stage.value.status === "marking" ? stage.value.id : null)),
    markingAll: computed(() => stage.value.status === "markingAll"),
    refresh,
    loadMore,
    markRead,
    markAllRead,
  };
}
