import { z } from "zod";

import { pageInfoSchema } from "./pages";

/**
 * WebAuthn returns browser-native binary fields, so these stay `unknown` until
 * `lib/webauthn.ts` casts them at the API boundary.
 */
export const passkeyOptionsSchema = z.looseObject({
  publicKey: z.record(z.string(), z.unknown()),
});
export type PasskeyOptions = z.infer<typeof passkeyOptionsSchema>;

/** Credential payload posted back to `/passkeys/register` and `/login/passkey`. */
export const passkeyCredentialSchema = z.record(z.string(), z.unknown());
export type PasskeyCredential = z.infer<typeof passkeyCredentialSchema>;

/** One registered credential shown on the passkeys page. */
export const passkeySummarySchema = z.looseObject({
  id: z.string(),
  name: z.string(),
  created_at: z.string(),
  last_used_at: z.string().optional(),
});
export type PasskeySummary = z.infer<typeof passkeySummarySchema>;

export const passkeysPageSchema = z.looseObject({
  passkeys: z.array(passkeySummarySchema).default([]),
  page: pageInfoSchema,
});
export type PasskeysPageProps = z.infer<typeof passkeysPageSchema>;
