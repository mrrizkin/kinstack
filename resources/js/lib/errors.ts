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
 * Inertia may deliver an array, and Vue casts an empty string to `true` for props
 * typed `[Boolean, String]`.
 */
export function fieldMessage(value: unknown): string | undefined {
  return firstMessage(value as ErrorValue);
}

/** Normalises an Inertia error bag for the dialog state machines. */
export function errorMap(value: Record<string, unknown> | undefined): Record<string, string> {
  const normalised: Record<string, string> = {};
  for (const [field, item] of Object.entries(value ?? {})) {
    const message = firstMessage(item as ErrorValue);
    if (message) {
      normalised[field] = message;
    }
  }
  return normalised;
}

export function useServerErrors() {
  const { errors } = usePageProps();

  // Local form state can hold arrays too, so normalise both sources.
  // Returning undefined instead of "" matters: UFormField declares error as
  // [Boolean, String], and Vue paints an empty string as an error with no message.
  const message = (field: string, local?: Record<string, unknown>): string | undefined =>
    firstMessage(errors.value[field]) || firstMessage(local?.[field] as ErrorValue | undefined);

  return {
    errors,
    message,
    formError: computed(() => message("form")),
  };
}
