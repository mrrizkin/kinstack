import { z } from "zod";

import { errorResponseSchema, type Problem } from "./contracts/page-props";

/**
 * Consistent request failure raised at the JSON boundary. Carries the HTTP
 * status and, when the server sent one, the stable `httperror.Problem` payload.
 */
export class RequestError extends Error {
  readonly status: number;
  readonly problem?: Problem;

  constructor(message: string, status: number, problem?: Problem) {
    super(message);
    this.name = "RequestError";
    this.status = status;
    this.problem = problem;
  }
}

export interface RequestJsonOptions {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  body?: unknown;
  csrfToken?: string;
  signal?: AbortSignal;
  headers?: Record<string, string>;
}

/**
 * Fetches a same-origin JSON endpoint, validates the payload and turns any
 * non-2xx response into a `RequestError`.
 *
 * Endpoints that answer with a redirect (and therefore HTML) should pass
 * `z.unknown()`, which accepts the empty payload after the browser follows the
 * redirect.
 */
export async function requestJson<T>(path: string, schema: z.ZodType<T>, options: RequestJsonOptions = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: "application/json", ...options.headers };
  if (options.csrfToken) {
    headers["X-XSRF-TOKEN"] = options.csrfToken;
  }

  let body: string | undefined;
  if (options.body !== undefined) {
    headers["Content-Type"] = "application/json";
    body = JSON.stringify(options.body);
  }

  const response = await fetch(path, {
    method: options.method ?? (body === undefined ? "GET" : "POST"),
    credentials: "same-origin",
    headers,
    body,
    signal: options.signal,
  });

  const text = await response.text();
  let payload: unknown;
  if (text.length > 0) {
    try {
      payload = JSON.parse(text);
    } catch {
      payload = undefined;
    }
  }

  if (!response.ok) {
    const parsed = errorResponseSchema.safeParse(payload);
    const problem = parsed.success ? parsed.data.error : undefined;
    throw new RequestError(problem?.message ?? `Request failed (${response.status}).`, response.status, problem);
  }

  return schema.parse(payload);
}
