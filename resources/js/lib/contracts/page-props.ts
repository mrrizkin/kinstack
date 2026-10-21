import { z } from "zod";

/**
 * Shared Inertia page props.
 *
 * These schemas describe values produced by the Go boundary. Parse them once at
 * the boundary (see `usePageProps`) instead of casting `page.props` per page.
 * Unknown keys are kept so the server can evolve without breaking the client.
 */
export const actorSchema = z.looseObject({
  public_id: z.string().optional(),
  username: z.string().optional(),
  display_name: z.string().optional(),
  role: z.string().optional(),
  superuser: z.boolean().optional(),
  scopes: z.array(z.string()).optional(),
});
export type Actor = z.infer<typeof actorSchema>;

export const authMethodsSchema = z.looseObject({
  magic_link: z.boolean().optional(),
  passkey: z.boolean().optional(),
  api_token: z.boolean().optional(),
  oauth: z.boolean().optional(),
});
export type AuthMethods = z.infer<typeof authMethodsSchema>;

export const errorBagSchema = z.record(
  z.string(),
  z.union([z.string(), z.array(z.string()), z.null()]).transform((value) => value ?? undefined),
);
export type ErrorBag = z.infer<typeof errorBagSchema>;

const glitchTipSchema = z.looseObject({
  dsn: z.string().optional(),
  environment: z.string().optional(),
  release: z.string().optional(),
  traces_sample_rate: z.number().optional(),
});

const umamiSchema = z.looseObject({
  enabled: z.boolean().optional(),
  script_url: z.string().optional(),
  website_id: z.string().optional(),
  domains: z.string().optional(),
  auto_track: z.boolean().optional(),
});

export const sharedPagePropsSchema = z.looseObject({
  app_name: z.string().optional(),
  csrf_token: z.string().optional(),
  media_enabled: z.boolean().optional(),
  auth_methods: authMethodsSchema.optional(),
  actor: actorSchema.nullish(),
  authz_permissions: z.array(z.string()).optional(),
  errors: errorBagSchema.optional(),
  observability: z.looseObject({ glitchtip: glitchTipSchema.optional() }).optional(),
  analytics: z.looseObject({ umami: umamiSchema.optional() }).optional(),
});
export type SharedPageProps = z.infer<typeof sharedPagePropsSchema>;

/** URL fragment carried on every Inertia page component. */
export const pageComponentSchema = z.string().min(1);

/** Public JSON error envelope (`httperror.Problem` under `error`). */
export const errorResponseSchema = z.object({
  error: z.looseObject({
    status: z.number().int(),
    code: z.string(),
    kind: z.string(),
    message: z.string(),
    retryable: z.boolean().optional(),
  }),
});
export type ErrorResponse = z.infer<typeof errorResponseSchema>;

export const problemSchema = errorResponseSchema.shape.error;
export type Problem = z.infer<typeof problemSchema>;
