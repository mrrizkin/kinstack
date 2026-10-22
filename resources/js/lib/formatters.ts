/** Formatting helpers shared across KinStack pages. */

export function formatDateTime(value: string | null | undefined): string {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  return new Intl.DateTimeFormat("en-US", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

/** Relative time used by the notification slide-over. */
export function formatRelativeTime(value: string | null | undefined): string {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }

  const formatter = new Intl.RelativeTimeFormat("en-US", { numeric: "auto" });
  const seconds = Math.round((date.getTime() - Date.now()) / 1000);
  const ranges: { limit: number; unit: Intl.RelativeTimeFormatUnit; divisor: number }[] = [
    { limit: 60, unit: "second", divisor: 1 },
    { limit: 3600, unit: "minute", divisor: 60 },
    { limit: 86400, unit: "hour", divisor: 3600 },
    { limit: 604800, unit: "day", divisor: 86400 },
    { limit: 2629800, unit: "week", divisor: 604800 },
    { limit: 31557600, unit: "month", divisor: 2629800 },
  ];

  const absolute = Math.abs(seconds);
  for (const range of ranges) {
    if (absolute < range.limit) {
      return formatter.format(Math.round(seconds / range.divisor), range.unit);
    }
  }
  return formatter.format(Math.round(seconds / 31557600), "year");
}
