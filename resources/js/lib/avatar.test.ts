import { describe, expect, it } from "bun:test";

import { avatarSeed, avatarUrl } from "./avatar";

describe("avatarSeed", () => {
  it("prefers the public id so a rename keeps the same avatar", () => {
    expect(avatarSeed({ public_id: "usr_1", username: "alex" })).toBe("usr_1");
  });

  it("falls back to the username when there is no public id", () => {
    expect(avatarSeed({ public_id: "", username: "alex" })).toBe("alex");
    expect(avatarSeed({ public_id: "   ", username: " alex " })).toBe("alex");
    expect(avatarSeed({ username: "alex" })).toBe("alex");
  });

  it("falls back to a stable seed for missing accounts", () => {
    expect(avatarSeed(null)).toBe("user");
    expect(avatarSeed({})).toBe("user");
    expect(avatarSeed({ public_id: " ", username: " " })).toBe("user");
  });
});

describe("avatarUrl", () => {
  it("uses a light background so the black artwork stays visible in dark mode", () => {
    const url = new URL(avatarUrl({ public_id: "usr_1" }));
    expect(url.searchParams.get("backgroundColor")).toBe("e5e7eb");
    expect(url.searchParams.get("backgroundColor")).not.toBe("transparent");
  });

  it("keeps seeds stable and bounds the requested size", () => {
    const url = new URL(avatarUrl({ public_id: "usr_1" }, 999));
    expect(url.searchParams.get("seed")).toBe("usr_1");
    expect(url.searchParams.get("size")).toBe("256");
  });

  it("never seeds from a display name", () => {
    const url = new URL(avatarUrl({ public_id: "usr_1", username: "alex" }));
    expect(url.searchParams.get("seed")).not.toBe("alex");
  });
});
