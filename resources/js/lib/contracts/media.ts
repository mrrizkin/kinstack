import { z } from "zod";

import { pageInfoSchema } from "./pages";

export const assetSchema = z.looseObject({
  public_id: z.string(),
  filename: z.string(),
  content_type: z.string(),
  size: z.number().int(),
  url: z.string(),
  created_at: z.string(),
});
export type Asset = z.infer<typeof assetSchema>;

export const mediaPageSchema = z.looseObject({
  assets: z.array(assetSchema).default([]),
  page: pageInfoSchema,
});
export type MediaPageProps = z.infer<typeof mediaPageSchema>;
