import { computed } from "vue";

import { usePageProps } from "./page-props";

type ErrorValue = string | string[] | undefined;

function firstMessage(value: ErrorValue): string | undefined {
  if (Array.isArray(value)) {
    return value.find((item) => item.length > 0) || undefined;
  }
  return value || undefined;
}

/**
 * Normalises an Inertia field error to the single string UFormField expects.
 * Inertia may deliver an array, and an empty string is unsafe because Vue casts
 * it to `true` for props typed `[Boolean, String]`.
 */
export function fieldMessage(value: unknown): string | undefined {
  return firstMessage(value as ErrorValue);
}

/**
 * Reads server-side (flashed) validation errors. Inertia delivers them as arrays,
 * so everything is normalised to a single string before rendering.
 */
export function useServerErrors() {
  const { errors } = usePageProps();

  // Local form state can hold arrays too, so normalise both sources.
  // Returning undefined instead of "" matters: UFormField declares error as
  // [Boolean, String], and Vue casts an empty string to true, which paints the
  // input red with no message.
  const message = (field: string, local?: Record<string, unknown>): string | undefined =>
    firstMessage(errors.value[field]) || firstMessage(local?.[field] as ErrorValue | undefined);

  return {
    errors,
    message,
    formError: computed(() => message("form")),
  };
}
