import { describe, expect, test } from "bun:test";

import { formatDate, formatDateTime, formatRelativeTime, formatRupiah } from "./formatters";

describe("formatters", () => {
  test("formats currency and invalid amounts", () => {
    expect(formatRupiah(15000)).toContain("15,000");
    expect(formatRupiah("not-a-number")).toMatch(/IDR\s0/);
    expect(formatRupiah(null)).toMatch(/IDR\s0/);
  });

  test("returns a dash for empty or invalid dates", () => {
    expect(formatDate(null)).toBe("-");
    expect(formatDate("not-a-date")).toBe("-");
    expect(formatDateTime("not-a-date")).toBe("-");
  });

  test("formats dates in English locale", () => {
    expect(formatDate("2026-10-20T00:00:00")).toContain("Oct 20, 2026");
    expect(formatDateTime("2026-10-20T03:00:00")).toContain("Oct 20, 2026");
  });

  test("formats relative time in English", () => {
    const past = new Date(Date.now() - 2 * 60 * 60 * 1000).toISOString();
    expect(formatRelativeTime(past)).toContain("hours ago");
  });
});
