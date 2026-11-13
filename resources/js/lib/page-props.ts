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

/**
 * Parses a page's own props at the Inertia boundary. `page.props` is replaced on
 * every visit while the component stays mounted, so each key is read through a
 * computed instead of freezing the first parse.
 */
export function usePageSchema<S extends z.ZodType>(schema: S): z.infer<S> {
  const page = usePage();
  const parsed = computed(() => schema.parse(page.props) as unknown as Record<string, unknown>);
  const props: Record<string, unknown> = {};

  for (const key of Object.keys(parsed.value)) {
    Object.defineProperty(props, key, {
      enumerable: true,
      get: () => parsed.value[key],
    });
  }

  return props as z.infer<S>;
}

/** Typed access to the shared Inertia props, parsed in one place. */
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
