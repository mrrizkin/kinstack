import { describe, expect, it } from "bun:test";

import { errorMap, fieldMessage } from "./errors";

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

describe("errorMap", () => {
  it("normalises every field and drops empty messages", () => {
    expect(errorMap({ name: ["Required"], rank: "", note: [""] })).toEqual({ name: "Required" });
  });

  it("treats a missing bag as no errors", () => {
    expect(errorMap(undefined)).toEqual({});
  });
});
