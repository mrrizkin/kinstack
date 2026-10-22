/**
 * Keeping the preset definitions out of the component makes date math and
 * matching unit-testable without mounting Vue.
 */

import { parseDate } from "@internationalized/date";

/** The audit selects use this for "no constraint". */
export const ALL_FILTER = "all";

export const RESET_FILTER_KEY = "reset";

export interface QuickFilterItem {
  key: string;
  label: string;
  active: boolean;
  icon?: string;
}

export interface AuditFilterState {
  actor_key: string;
  action: string;
  outcome: string;
  search: string;
  from: string;
  to: string;
}

export interface AuditPreset {
  key: string;
  label: string;
  values: Partial<AuditFilterState>;
}

export interface AuditPresetOptions {
  /** Current UTC day as `YYYY-MM-DD`; injected so tests are deterministic. */
  today: string;
  /** The signed-in actor's public id, used by the "My activity" preset. */
  actorKey?: string;
}

/** Adds days to a `YYYY-MM-DD` string. Calendar arithmetic, so no timezone drift. */
export function shiftUtcDate(date: string, days: number): string {
  return parseDate(date).add({ days }).toString();
}

/** Today as a UTC `YYYY-MM-DD` string, matching how audit parses date filters. */
export function utcToday(now: Date = new Date()): string {
  return now.toISOString().slice(0, 10);
}

export function createAuditPresets({ today, actorKey = "" }: AuditPresetOptions): AuditPreset[] {
  const presets: AuditPreset[] = [
    { key: "last_24h", label: "Last 24 hours", values: { from: shiftUtcDate(today, -1), to: today } },
    { key: "last_7d", label: "Last 7 days", values: { from: shiftUtcDate(today, -6), to: today } },
    { key: "last_30d", label: "Last 30 days", values: { from: shiftUtcDate(today, -29), to: today } },
    { key: "failures", label: "Failures", values: { outcome: "failure" } },
    { key: "changes", label: "Changes", values: { action: "http.mutate" } },
    { key: "page_views", label: "Page views", values: { action: "http.read" } },
  ];
  if (actorKey) {
    presets.push({ key: "mine", label: "My activity", values: { actor_key: actorKey } });
  }
  return presets;
}

export function isPresetActive(preset: AuditPreset, state: AuditFilterState): boolean {
  return Object.entries(preset.values).every(([key, value]) => state[key as keyof AuditFilterState] === value);
}

/**
 * Reset trails the presets so their position stays stable, and it only exists
 * while a filter is applied.
 */
export function createQuickFilterItems(presets: AuditPreset[], state: AuditFilterState, resetIcon: string) {
  const items: QuickFilterItem[] = presets.map((preset) => ({
    key: preset.key,
    label: preset.label,
    active: isPresetActive(preset, state),
  }));
  if (Object.keys(state).some((key) => isFilterSet(state, key as keyof AuditFilterState))) {
    items.push({ key: RESET_FILTER_KEY, label: "Reset", active: false, icon: resetIcon });
  }
  return items;
}

function isFilterSet(state: AuditFilterState, key: keyof AuditFilterState): boolean {
  return state[key] !== "" && state[key] !== ALL_FILTER;
}

/** The released value for a key: selects fall back to the "all" sentinel. */
function resetValue(key: keyof AuditFilterState): string {
  return key === "action" || key === "outcome" ? ALL_FILTER : "";
}

/**
 * Toggles a preset: active presets release the keys they own, inactive ones
 * merge in their values. Keys the preset does not own are never touched, so a
 * quick filter composes with whatever the user already set.
 */
export function togglePreset(state: AuditFilterState, preset: AuditPreset): AuditFilterState {
  const next = { ...state };
  const active = isPresetActive(preset, state);
  for (const key of Object.keys(preset.values) as (keyof AuditFilterState)[]) {
    next[key] = active ? resetValue(key) : (preset.values[key] as string);
  }
  return next;
}
