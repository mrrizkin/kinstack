import { describe, expect, test } from "bun:test";

import { fromCalendarDate, toCalendarDate } from "./date";

describe("date helpers", () => {
  test("parses ISO dates into calendar values", () => {
    expect(toCalendarDate("2026-10-20")?.toString()).toBe("2026-10-20");
    expect(toCalendarDate("")).toBeUndefined();
    expect(toCalendarDate("invalid")).toBeUndefined();
  });

  test("serializes calendar values back to ISO dates", () => {
    const value = toCalendarDate("2026-10-20");
    expect(fromCalendarDate(value)).toBe("2026-10-20");
    expect(fromCalendarDate(null)).toBe("");
    expect(fromCalendarDate({ start: value, end: value })).toBe("");
  });
});
