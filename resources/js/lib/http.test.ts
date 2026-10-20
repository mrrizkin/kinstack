import { afterEach, describe, expect, test } from "bun:test";

import { z } from "zod";

import { RequestError, requestJson } from "./http";

const originalFetch = globalThis.fetch;

function respond(body: string, status = 200) {
  return new Response(body, { status, headers: { "Content-Type": "application/json" } });
}

afterEach(() => {
  globalThis.fetch = originalFetch;
});

describe("requestJson", () => {
  test("parses a successful payload with the schema", async () => {
    globalThis.fetch = (async () => respond(JSON.stringify({ unread: 3 }))) as typeof fetch;

    const result = await requestJson("/notifications/feed", z.object({ unread: z.number() }));
    expect(result.unread).toBe(3);
  });

  test("sends the CSRF header and JSON body", async () => {
    let seen: RequestInit | undefined;
    globalThis.fetch = (async (_input: RequestInfo | URL, init?: RequestInit) => {
      seen = init;
      return respond(JSON.stringify({ ok: true }));
    }) as typeof fetch;

    await requestJson("/passkeys/register", z.object({ ok: z.boolean() }), {
      method: "POST",
      csrfToken: "csrf-value",
      body: { name: "Laptop" },
    });

    const headers = seen?.headers as Record<string, string>;
    expect(headers["X-XSRF-TOKEN"]).toBe("csrf-value");
    expect(headers["Content-Type"]).toBe("application/json");
    expect(seen?.body).toBe(JSON.stringify({ name: "Laptop" }));
    expect(seen?.credentials).toBe("same-origin");
  });

  test("throws a RequestError carrying the server problem", async () => {
    globalThis.fetch = (async () =>
      respond(
        JSON.stringify({
          error: { status: 422, code: "invalid_input", kind: "validation", message: "Name is required" },
        }),
        422,
      )) as typeof fetch;

    const failure = requestJson("/passkeys/register", z.unknown());
    await expect(failure).rejects.toBeInstanceOf(RequestError);
    await expect(failure).rejects.toMatchObject({
      status: 422,
      message: "Name is required",
      problem: { code: "invalid_input" },
    });
  });

  test("throws a generic RequestError for a non-JSON failure", async () => {
    globalThis.fetch = (async () => new Response("<html>nope</html>", { status: 500 })) as typeof fetch;

    await expect(requestJson("/anything", z.unknown())).rejects.toMatchObject({
      status: 500,
      message: "Request failed (500).",
    });
  });

  test("accepts an empty body for redirect endpoints", async () => {
    globalThis.fetch = (async () => new Response("", { status: 200 })) as typeof fetch;
    await expect(requestJson("/notifications/read-all", z.unknown(), { method: "POST" })).resolves.toBeUndefined();
  });

  test("rejects a payload that does not match the schema", async () => {
    globalThis.fetch = (async () => respond(JSON.stringify({ unread: "three" }))) as typeof fetch;
    await expect(requestJson("/notifications/feed", z.object({ unread: z.number() }))).rejects.toThrow();
  });
});
