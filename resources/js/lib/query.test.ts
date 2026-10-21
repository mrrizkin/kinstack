import { describe, expect, test } from "bun:test";

import { queryString } from "./query";

describe("queryString", () => {
  test("omits empty values", () => {
    expect(queryString({ page: undefined, search: "", filter: null })).toBe("");
  });

  test("keeps zero and false as values", () => {
    expect(queryString({ page: 0, active: false })).toBe("?page=0&active=false");
  });

  test("encodes values and includes a leading question mark", () => {
    const result = queryString({ page: 2, search: "hello world" });
    expect(result).toBe("?page=2&search=hello+world");
    expect(new URLSearchParams(result).get("search")).toBe("hello world");
  });
});
