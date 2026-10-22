import { describe, expect, test } from "bun:test";

import { DEFAULT_PAGE_SIZE, PAGE_SIZES, isPageSize, pageSizeOptions } from "./pagination";

describe("page sizes", () => {
  test("defaults to the smallest offered size", () => {
    expect(DEFAULT_PAGE_SIZE).toBe(PAGE_SIZES[0]);
  });

  test("recognises offered sizes only", () => {
    expect(isPageSize(5)).toBe(true);
    expect(isPageSize(100)).toBe(true);
    expect(isPageSize(7)).toBe(false);
    expect(isPageSize(0)).toBe(false);
  });

  test("builds options in ascending order", () => {
    expect(pageSizeOptions(10).map((option) => option.value)).toEqual(PAGE_SIZES.map(String));
  });

  test("keeps a legacy size selectable", () => {
    const values = pageSizeOptions(7).map((option) => option.value);
    expect(values).toEqual(["5", "7", "10", "25", "50", "100"]);
  });
});
