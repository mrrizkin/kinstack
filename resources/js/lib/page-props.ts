import { usePage } from "@inertiajs/vue3";
import { computed } from "vue";
import type { z } from "zod";

import {
  authMethodsSchema,
  errorBagSchema,
  sharedPagePropsSchema,
  type Actor,
  type AuthMethods,
  type ErrorBag,
  type SharedPageProps,
} from "./contracts/page-props";

/** Parses a page's own props at the Inertia boundary and returns the typed copy. */
export function usePageSchema<S extends z.ZodType>(schema: S): z.infer<S> {
  return schema.parse(usePage().props);
}

/**
 * Typed access to the shared Inertia props.
 *
 * This is the single place where `page.props` is parsed. Pages and layouts read
 * these computed values instead of casting the raw prop bag themselves.
 */
export function usePageProps() {
  const page = usePage();

  const shared = computed<SharedPageProps>(() => sharedPagePropsSchema.parse(page.props));
  const actor = computed<Actor | null>(() => shared.value.actor ?? null);
  const authMethods = computed<AuthMethods>(() => authMethodsSchema.parse(shared.value.auth_methods ?? {}));
  const permissions = computed<string[]>(() => shared.value.authz_permissions ?? []);
  const csrfToken = computed<string>(() => shared.value.csrf_token ?? "");
  const errors = computed<ErrorBag>(() => errorBagSchema.parse(shared.value.errors ?? {}));
  const mediaEnabled = computed<boolean>(() => shared.value.media_enabled ?? false);

  return { page, shared, actor, authMethods, permissions, csrfToken, errors, mediaEnabled };
}
