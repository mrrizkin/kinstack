import { describe, expect, test } from "bun:test";

import { icons } from "./icons";

const collection = "i-material-symbols:";

describe("icons", () => {
  test("uses one collection so the set can be migrated in one file", () => {
    for (const name of Object.values(icons)) {
      expect(name.startsWith(collection)).toBe(true);
    }
  });
});
