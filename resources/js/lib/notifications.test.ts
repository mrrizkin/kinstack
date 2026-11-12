import { describe, expect, test } from "bun:test";

import type { NotificationItem } from "./contracts/notifications";
import type { RequestJsonOptions } from "./http";
import { notificationErrorKey, useNotificationFeed } from "./notifications";

type Handler = (path: string, options?: RequestJsonOptions) => unknown;

interface FeedResponse {
  notifications: NotificationItem[];
  unread: number;
  next_cursor: string;
}

function item(id: string, read = false): NotificationItem {
  return {
    public_id: id,
    kind: "security",
    title: `Notification ${id}`,
    body: "Body",
    read,
    created_at: "2026-01-01T00:00:00Z",
  };
}

function feed(payload: FeedResponse) {
  return payload;
}

function fakeRequest(handler: Handler) {
  // Matches the injected boundary: parse whatever the handler returns.
  const request = async <T>(
    path: string,
    schema: { parse(value: unknown): T },
    options?: RequestJsonOptions,
  ): Promise<T> => schema.parse(handler(path, options));
  return request;
}

describe("notification feed", () => {
  test("refresh seeds the list, the cursor, and the unread badge", async () => {
    const seen: number[] = [];
    const feedState = useNotificationFeed({
      csrfToken: () => "token",
      onUnread: (unread) => seen.push(unread),
      request: fakeRequest(() => feed({ notifications: [item("a")], unread: 1, next_cursor: "next" })),
    });

    await feedState.refresh();

    expect(feedState.notifications.value.map((row) => row.public_id)).toEqual(["a"]);
    expect(feedState.unread.value).toBe(1);
    expect(feedState.nextCursor.value).toBe("next");
    expect(seen).toEqual([1]);
    expect(feedState.busy.value).toBe(false);
  });

  test("loadMore appends without duplicating and follows the cursor", async () => {
    const feedState = useNotificationFeed({
      csrfToken: () => "token",
      initial: { notifications: [item("a")], unread: 1, nextCursor: "cursor-1" },
      request: fakeRequest((path) => {
        expect(path).toContain("cursor=cursor-1");
        return feed({ notifications: [item("b")], unread: 2, next_cursor: "" });
      }),
    });

    await feedState.loadMore();

    expect(feedState.notifications.value.map((row) => row.public_id)).toEqual(["a", "b"]);
    expect(feedState.nextCursor.value).toBe("");
  });

  test("loadMore without a cursor does nothing", async () => {
    let calls = 0;
    const feedState = useNotificationFeed({
      csrfToken: () => "token",
      initial: { nextCursor: "" },
      request: fakeRequest(() => {
        calls += 1;
        return feed({ notifications: [], unread: 0, next_cursor: "" });
      }),
    });

    await feedState.loadMore();
    expect(calls).toBe(0);
  });

  test("markRead posts once and decrements the unread count", async () => {
    const calls: string[] = [];
    const read = item("a");
    const feedState = useNotificationFeed({
      csrfToken: () => "token",
      initial: { notifications: [read], unread: 1, nextCursor: "next" },
      request: fakeRequest((path) => {
        calls.push(path);
        return undefined;
      }),
    });

    await feedState.markRead(read);
    await feedState.markRead(read);

    expect(calls).toEqual(["/notifications/a/read"]);
    expect(read.read).toBe(true);
    expect(feedState.unread.value).toBe(0);
  });

  test("markAllRead marks every row and zeroes the badge", async () => {
    const feedState = useNotificationFeed({
      csrfToken: () => "token",
      initial: { notifications: [item("a"), item("b")], unread: 2, nextCursor: "next" },
      request: fakeRequest(() => undefined),
    });

    await feedState.markAllRead();

    expect(feedState.notifications.value.every((row) => row.read)).toBe(true);
    expect(feedState.unread.value).toBe(0);
  });

  test("a failed request becomes a typed error and returns to idle", async () => {
    const feedState = useNotificationFeed({
      csrfToken: () => "token",
      request: fakeRequest(() => {
        throw new Error("offline");
      }),
    });

    await feedState.refresh();

    expect(feedState.error.value).toBe("load");
    expect(feedState.busy.value).toBe(false);
    expect(notificationErrorKey("load")).toBe("shell.notifications.loadError");
    expect(notificationErrorKey("mark")).toBe("shell.notifications.markError");
    expect(notificationErrorKey("markAll")).toBe("shell.notifications.markAllError");
  });

  test("a second action while busy is ignored", async () => {
    let release: () => void = () => {};
    const gate = new Promise<void>((resolve) => {
      release = resolve;
    });
    let calls = 0;
    const request = async <T>(_path: string, schema: { parse(value: unknown): T }): Promise<T> => {
      calls += 1;
      await gate;
      return schema.parse({ notifications: [], unread: 0, next_cursor: "" });
    };
    const guarded = useNotificationFeed({
      csrfToken: () => "token",
      initial: { nextCursor: "cursor-1" },
      request,
    });

    const first = guarded.loadMore();
    const second = guarded.loadMore();
    expect(guarded.loadingMore.value).toBe(true);

    release();
    await Promise.all([first, second]);

    expect(calls).toBe(1);
    expect(guarded.busy.value).toBe(false);
  });
});
