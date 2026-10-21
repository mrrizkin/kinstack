import { describe, expect, it } from "bun:test";

import { avatarUrl } from "./avatar";

describe("avatarUrl", () => {
  it("uses a light background so the black artwork stays visible in dark mode", () => {
    const url = new URL(avatarUrl("mrrizkin"));
    expect(url.searchParams.get("backgroundColor")).toBe("e5e7eb");
    expect(url.searchParams.get("backgroundColor")).not.toBe("transparent");
  });

  it("keeps seeds stable and bounds the requested size", () => {
    const url = new URL(avatarUrl("  mrrizkin  ", 999));
    expect(url.searchParams.get("seed")).toBe("mrrizkin");
    expect(url.searchParams.get("size")).toBe("256");
  });

  it("falls back to a stable seed for missing accounts", () => {
    const url = new URL(avatarUrl(null));
    expect(url.searchParams.get("seed")).toBe("user");
  });
});
