export type QueryValue = string | number | boolean | null | undefined;

/**
 * Builds a query string for a URL. Empty values are omitted so callers can pass
 * optional filters without assembling `?key=value` by hand.
 */
export function queryString(values: Record<string, QueryValue>): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(values)) {
    if (value === null || value === undefined || value === "") continue;
    params.set(key, String(value));
  }
  const query = params.toString();
  return query ? `?${query}` : "";
}
