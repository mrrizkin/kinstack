import { describe, expect, test } from "bun:test";

import { formatDateTime, formatRelativeTime } from "./formatters";

describe("formatters", () => {
  test("returns a dash for empty or invalid dates", () => {
    expect(formatDateTime("not-a-date")).toBe("-");
    expect(formatDateTime(null)).toBe("-");
  });

  test("formats dates in English locale", () => {
    expect(formatDateTime("2026-10-20T03:00:00")).toContain("Oct 20, 2026");
  });

  test("formats relative time in English", () => {
    const past = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeTime(past)).toContain("hours ago");
  });
});
