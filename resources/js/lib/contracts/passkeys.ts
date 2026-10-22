import { z } from "zod";

import { pageInfoSchema } from "./pages";

/**
 * WebAuthn the browser gives us a `PublicKeyCredential`, but the Go boundary
 * only declares the generic `{ publicKey: object }` envelope. The decoded
 * binary fields are browser-native, so they stay `unknown` until the WebAuthn
 * API consumes them (see `lib/webauthn.ts` for the documented casts).
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
