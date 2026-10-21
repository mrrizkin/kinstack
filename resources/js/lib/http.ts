import { http, HttpResponseError } from "@inertiajs/core";
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
  headers?: Record<string, string>;
  signal?: AbortSignal;
}

function parsePayload(raw: string | null | undefined): unknown {
  if (!raw) return undefined;
  try {
    return JSON.parse(raw);
  } catch {
    return undefined;
  }
}

function failure(status: number, payload: unknown): RequestError {
  const parsed = errorResponseSchema.safeParse(payload);
  const problem = parsed.success ? parsed.data.error : undefined;
  return new RequestError(problem?.message ?? `Request failed (${status}).`, status, problem);
}

/**
 * Sends a same-origin JSON request through Inertia's HTTP client, validates the
 * payload and turns any non-2xx response into a `RequestError`.
 *
 * Endpoints that answer with a redirect (and therefore HTML) should pass
 * `z.unknown()`, which accepts the empty payload after the browser follows the
 * redirect.
 */
export async function requestJson<T>(path: string, schema: z.ZodType<T>, options: RequestJsonOptions = {}): Promise<T> {
  const verbs = { GET: "get", POST: "post", PUT: "put", PATCH: "patch", DELETE: "delete" } as const;
  const method = verbs[options.method ?? (options.body === undefined ? "GET" : "POST")];
  const headers: Record<string, string> = { Accept: "application/json", ...options.headers };
  // The CSRF cookie is HttpOnly, so script cannot read it; the token has to
  // travel in the header the server's extractor expects.
  if (options.csrfToken) {
    headers["X-XSRF-TOKEN"] = options.csrfToken;
  }

  const data = options.body === undefined ? undefined : JSON.stringify(options.body);
  if (data !== undefined && headers["Content-Type"] === undefined) {
    headers["Content-Type"] = "application/json";
  }

  try {
    const response = await http.getClient().request({ method, url: path, data, headers, signal: options.signal });
    return schema.parse(parsePayload(response.data));
  } catch (error) {
    if (error instanceof HttpResponseError) {
      throw failure(error.response.status, parsePayload(error.response.data));
    }
    throw error;
  }
}
