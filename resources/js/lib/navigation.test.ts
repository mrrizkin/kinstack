import { describe, expect, test } from "bun:test";

import { isNavigationItemVisible } from "./navigation";

const can = (code: string) => code === "media.view";

describe("navigation visibility", () => {
  test("shows entries without a permission or feature gate", () => {
    expect(isNavigationItemVisible({}, can, false)).toBe(true);
  });

  test("hides entries the actor cannot access", () => {
    expect(isNavigationItemVisible({ permission: "users.view" }, can, true)).toBe(false);
  });

  test("hides feature-gated entries when the feature is disabled", () => {
    expect(isNavigationItemVisible({ permission: "media.view", requiresMedia: true }, can, false)).toBe(false);
  });

  test("shows feature-gated entries when the feature is enabled and permitted", () => {
    expect(isNavigationItemVisible({ permission: "media.view", requiresMedia: true }, can, true)).toBe(true);
  });
});
