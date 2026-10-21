import { afterEach, describe, expect, test } from "bun:test";

import { http, type HttpClient, type HttpRequestConfig, type HttpResponse, HttpResponseError } from "@inertiajs/core";
import { z } from "zod";

import { RequestError, requestJson } from "./http";

const defaultClient = http.getClient();
let seen: HttpRequestConfig | undefined;

function json(body: unknown, status = 200): HttpResponse {
  return { status, data: typeof body === "string" ? body : JSON.stringify(body), headers: {} };
}

function useClient(handler: (config: HttpRequestConfig) => HttpResponse) {
  const client: HttpClient = {
    async request(config) {
      seen = config;
      const response = handler(config);
      if (response.status >= 400) {
        throw new HttpResponseError(`Request failed with status ${response.status}`, response, config.url);
      }
      return response;
    },
  };
  http.setClient(client);
}

afterEach(() => {
  http.setClient(defaultClient);
  seen = undefined;
});

describe("requestJson", () => {
  test("parses a successful payload with the schema", async () => {
    useClient(() => json({ unread: 3 }));

    const result = await requestJson("/notifications/feed", z.object({ unread: z.number() }));
    expect(result.unread).toBe(3);
    expect(seen?.method).toBe("get");
    expect(seen?.data).toBeUndefined();
  });

  test("sends the CSRF header, JSON body and abort signal", async () => {
    useClient(() => json({ ok: true }));
    const controller = new AbortController();

    await requestJson("/passkeys/register", z.object({ ok: z.boolean() }), {
      method: "POST",
      csrfToken: "csrf-value",
      body: { name: "Laptop" },
      signal: controller.signal,
    });

    expect(seen?.headers?.["X-XSRF-TOKEN"]).toBe("csrf-value");
    expect(seen?.headers?.["Content-Type"]).toBe("application/json");
    expect(seen?.data).toBe(JSON.stringify({ name: "Laptop" }));
    expect(seen?.method).toBe("post");
    expect(seen?.signal).toBe(controller.signal);
  });

  test("throws a RequestError carrying the server problem", async () => {
    useClient(() =>
      json({ error: { status: 422, code: "invalid_input", kind: "validation", message: "Name is required" } }, 422),
    );

    await expect(requestJson("/passkeys/register", z.unknown())).rejects.toMatchObject({
      status: 422,
      message: "Name is required",
      problem: { code: "invalid_input" },
    });
  });

  test("reports a 422 even when the schema accepts anything", async () => {
    useClient(() =>
      json({ error: { status: 422, code: "invalid_input", kind: "validation", message: "Name is required" } }, 422),
    );

    const failure = requestJson("/notifications/read-all", z.unknown(), { method: "POST" });
    await expect(failure).rejects.toBeInstanceOf(RequestError);
  });

  test("throws a generic RequestError for a non-JSON failure", async () => {
    useClient(() => json("<html>nope</html>", 500));

    await expect(requestJson("/anything", z.unknown())).rejects.toMatchObject({
      status: 500,
      message: "Request failed (500).",
    });
  });

  test("accepts an empty body for redirect endpoints", async () => {
    useClient(() => json(""));
    await expect(requestJson("/notifications/read-all", z.unknown(), { method: "POST" })).resolves.toBeUndefined();
  });

  test("rejects a payload that does not match the schema", async () => {
    useClient(() => json({ unread: "three" }));
    await expect(requestJson("/notifications/feed", z.object({ unread: z.number() }))).rejects.toThrow();
  });

  test("propagates transport errors unchanged", async () => {
    useClient(() => {
      throw new TypeError("network down");
    });
    await expect(requestJson("/anything", z.unknown())).rejects.toBeInstanceOf(TypeError);
  });
});
