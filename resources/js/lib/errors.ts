import { computed } from "vue";

import { usePageProps } from "./page-props";

type ErrorValue = string | string[] | undefined;

function firstMessage(value: ErrorValue): string {
  if (Array.isArray(value)) {
    return value.find((item) => item.length > 0) ?? "";
  }
  return typeof value === "string" ? value : "";
}

/**
 * Reads server-side (flashed) validation errors. Inertia delivers them as arrays,
 * so everything is normalised to a single string before rendering.
 */
export function useServerErrors() {
  const { errors } = usePageProps();

  // Local form state can hold arrays too, so normalise both sources.
  const message = (field: string, local?: Record<string, unknown>): string =>
    firstMessage(errors.value[field]) || firstMessage(local?.[field] as ErrorValue | undefined);

  return {
    errors,
    message,
    formError: computed(() => message("form")),
  };
}
