/**
 * Deterministic avatar helper backed by the DiceBear HTTP API.
 *
 * The seed is stable per account so the same user always gets the same image.
 * Size is capped to keep the request (and the rendered SVG) small.
 */
const DICEBEAR_ENDPOINT = "https://api.dicebear.com/9.x/notionists-neutral/svg";
const MAX_SIZE = 256;
// The notionists-neutral artwork is mostly black line work. A transparent
// background disappears in dark mode, so give every avatar a light surface.
const AVATAR_BACKGROUND = "e5e7eb";

export function avatarUrl(seed: string | null | undefined, size = 64): string {
  const normalizedSeed = (seed ?? "").trim() || "user";
  const boundedSize = Math.min(Math.max(Math.round(size), 16), MAX_SIZE);
  const params = new URLSearchParams({
    seed: normalizedSeed,
    size: String(boundedSize),
    backgroundColor: AVATAR_BACKGROUND,
  });
  return `${DICEBEAR_ENDPOINT}?${params.toString()}`;
}
