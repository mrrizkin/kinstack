import { CalendarDate, parseDate } from "@internationalized/date";

/** Converts a `YYYY-MM-DD` string into the value UInputDate/UCalendar expects. */
export function toCalendarDate(value: string | null | undefined): CalendarDate | undefined {
  if (!value) return undefined;
  try {
    return parseDate(value);
  } catch {
    return undefined;
  }
}

/** Converts a UInputDate value back into a `YYYY-MM-DD` string. */
export function fromCalendarDate(value: unknown): string {
  if (!value || Array.isArray(value)) return "";
  if (typeof value !== "object") return "";
  const text = String(value);
  return /^\d{4}-\d{2}-\d{2}/.test(text) ? text.slice(0, 10) : "";
}

/** A date range as two `YYYY-MM-DD` strings, used by the range date picker. */
export interface DateRangeInput {
  start: string;
  end: string;
}
