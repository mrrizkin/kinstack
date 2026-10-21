import { describe, expect, it } from "bun:test";

import { fieldMessage } from "./errors";

describe("fieldMessage", () => {
  it("returns the first non-empty server message", () => {
    expect(fieldMessage(["", "Required"])).toBe("Required");
    expect(fieldMessage("Required")).toBe("Required");
  });

  it("returns undefined instead of an empty string", () => {
    expect(fieldMessage("")).toBeUndefined();
    expect(fieldMessage([])).toBeUndefined();
    expect(fieldMessage(undefined)).toBeUndefined();
    expect(fieldMessage(null)).toBeUndefined();
  });
});
