import { z } from "zod";

export const pageInfoSchema = z.looseObject({
  page: z.number().int(),
  per_page: z.number().int(),
  total: z.number().int(),
  total_pages: z.number().int(),
  from: z.number().int(),
  to: z.number().int(),
  has_prev: z.boolean(),
  has_next: z.boolean(),
});
export type PageInfo = z.infer<typeof pageInfoSchema>;

/** Rendered by the 403 / maintenance / offline / help surfaces. */
export const systemStatePageSchema = z.looseObject({
  surface: z.string(),
  topics: z.array(z.string()).default([]),
  shortcuts: z.array(z.looseObject({ key: z.string(), action: z.string() })).default([]),
});
export type SystemStatePageProps = z.infer<typeof systemStatePageSchema>;

export const errorPageSchema = z.looseObject({
  status: z.number().optional(),
  message: z.string().optional(),
  code: z.string().optional(),
});
export type ErrorPageProps = z.infer<typeof errorPageSchema>;

export const loginPageSchema = z.looseObject({
  notice: z.string().optional(),
  oauth_enabled: z.boolean().optional(),
  oauth_providers: z.array(z.looseObject({ name: z.string(), label: z.string() })).optional(),
  oauth_error: z.string().optional(),
  magic_link_enabled: z.boolean().optional(),
  passkey_enabled: z.boolean().optional(),
  password_reset_enabled: z.boolean().optional(),
  saml_enabled: z.boolean().optional(),
});
export type LoginPageProps = z.infer<typeof loginPageSchema>;

export const forgotPasswordPageSchema = z.looseObject({
  notice: z.string().optional(),
});
export type ForgotPasswordPageProps = z.infer<typeof forgotPasswordPageSchema>;

export const resetPasswordPageSchema = z.looseObject({
  token: z.string().optional(),
  done: z.boolean().optional(),
});
export type ResetPasswordPageProps = z.infer<typeof resetPasswordPageSchema>;
