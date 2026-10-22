import { describe, expect, test } from "bun:test";

import {
  ALL_FILTER,
  createAuditPresets,
  createQuickFilterItems,
  isPresetActive,
  shiftUtcDate,
  togglePreset,
  utcToday,
  type AuditFilterState,
  type AuditPreset,
} from "./presets";

const base: AuditFilterState = {
  actor_key: "",
  action: ALL_FILTER,
  outcome: ALL_FILTER,
  search: "",
  from: "",
  to: "",
};

function preset(key: string, actorKey = "actor-1"): AuditPreset {
  const found = createAuditPresets({ today: "2026-03-15", actorKey }).find((item) => item.key === key);
  if (!found) throw new Error(`missing preset ${key}`);
  return found;
}

describe("shiftUtcDate", () => {
  test("shifts across month and year boundaries", () => {
    expect(shiftUtcDate("2026-03-15", -6)).toBe("2026-03-09");
    expect(shiftUtcDate("2026-03-15", -29)).toBe("2026-02-14");
    expect(shiftUtcDate("2026-01-01", -1)).toBe("2025-12-31");
  });
});

describe("utcToday", () => {
  test("formats the current day in UTC", () => {
    expect(utcToday(new Date("2026-03-15T23:30:00Z"))).toBe("2026-03-15");
  });
});

describe("createAuditPresets", () => {
  test("builds UTC ranges ending today", () => {
    const presets = createAuditPresets({ today: "2026-03-15", actorKey: "actor-1" });
    expect(presets.find((item) => item.key === "last_7d")?.values).toEqual({ from: "2026-03-09", to: "2026-03-15" });
    expect(presets.find((item) => item.key === "last_30d")?.values).toEqual({ from: "2026-02-14", to: "2026-03-15" });
    expect(presets.find((item) => item.key === "last_24h")?.values).toEqual({ from: "2026-03-14", to: "2026-03-15" });
  });

  test("omits the actor preset when there is no actor", () => {
    const presets = createAuditPresets({ today: "2026-03-15" });
    expect(presets.some((item) => item.key === "mine")).toBe(false);
    expect(createAuditPresets({ today: "2026-03-15", actorKey: "actor-1" }).some((item) => item.key === "mine")).toBe(
      true,
    );
  });
});

describe("isPresetActive", () => {
  test("requires an exact match of every owned value", () => {
    expect(isPresetActive(preset("changes"), base)).toBe(false);
    expect(isPresetActive(preset("changes"), { ...base, action: "http.mutate" })).toBe(true);
    expect(isPresetActive(preset("changes"), { ...base, action: "http.mutate", search: "login" })).toBe(true);
  });

  test("ignores manual date ranges that differ from the preset range", () => {
    expect(isPresetActive(preset("last_7d"), { ...base, from: "2026-01-01", to: "2026-03-15" })).toBe(false);
  });
});

describe("createQuickFilterItems", () => {
  const presets = createAuditPresets({ today: "2026-03-15", actorKey: "actor-1" });

  test("marks only the matching presets active", () => {
    const items = createQuickFilterItems(presets, { ...base, outcome: "failure" }, "reset-icon");
    expect(items.filter((item) => item.active).map((item) => item.key)).toEqual(["failures"]);
  });

  test("hides reset while nothing is applied", () => {
    expect(createQuickFilterItems(presets, base, "reset-icon").some((item) => item.key === "reset")).toBe(false);
  });

  test("appends reset last once a filter is applied", () => {
    const items = createQuickFilterItems(presets, { ...base, search: "login" }, "reset-icon");
    const reset = items.at(-1);
    expect(reset?.key).toBe("reset");
    expect(reset?.icon).toBe("reset-icon");
    expect(reset?.active).toBe(false);
  });
});

describe("togglePreset", () => {
  test("merges preset values into the filter", () => {
    const next = togglePreset(base, preset("failures"));
    expect(next).toEqual({ ...base, outcome: "failure" });
  });

  test("replaces the range owned by another time preset", () => {
    const next = togglePreset({ ...base, from: "2026-03-01", to: "2026-03-15" }, preset("last_7d"));
    expect(next).toEqual({ ...base, from: "2026-03-09", to: "2026-03-15" });
  });

  test("releases only its own keys when toggled off", () => {
    const state = { ...base, action: "http.mutate", search: "login", from: "2026-03-09", to: "2026-03-15" };
    const next = togglePreset(state, preset("changes"));
    expect(next).toEqual({ ...state, action: ALL_FILTER });
  });

  test("composes independent presets", () => {
    const state = togglePreset(togglePreset(base, preset("last_7d")), preset("failures"));
    expect(state.from).toBe("2026-03-09");
    expect(state.outcome).toBe("failure");
  });

  test("scopes the actor preset to the signed-in actor", () => {
    const state = togglePreset(base, preset("mine", "actor-9"));
    expect(state.actor_key).toBe("actor-9");
    expect(togglePreset(state, preset("mine", "actor-9")).actor_key).toBe("");
  });
});
