/**
 * Deterministic avatar helper backed by the DiceBear HTTP API.
 *
 * The seed is the account's public id, which never changes; username only
 * covers actors without a profile row. Size is capped to keep the request
 * (and the rendered SVG) small.
 */
const DICEBEAR_ENDPOINT = "https://api.dicebear.com/9.x/notionists-neutral/svg";
const MAX_SIZE = 256;
// The notionists-neutral artwork is mostly black line work. A transparent
// background disappears in dark mode, so give every avatar a light surface.
const AVATAR_BACKGROUND = "e5e7eb";
const FALLBACK_SEED = "user";

/** Identity fields an avatar seed is derived from, in priority order. */
export interface AvatarSeed {
  public_id?: string | null;
  username?: string | null;
}

export function avatarSeed(seed: AvatarSeed | null | undefined): string {
  return (seed?.public_id ?? "").trim() || (seed?.username ?? "").trim() || FALLBACK_SEED;
}

export function avatarUrl(seed: AvatarSeed | null | undefined, size = 64): string {
  const boundedSize = Math.min(Math.max(Math.round(size), 16), MAX_SIZE);
  const params = new URLSearchParams({
    seed: avatarSeed(seed),
    size: String(boundedSize),
    backgroundColor: AVATAR_BACKGROUND,
  });
  return `${DICEBEAR_ENDPOINT}?${params.toString()}`;
}
