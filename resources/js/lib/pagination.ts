/**
 * Adding a size is a three-way change: here, `internal/pkg/pagination`, and the
 * `user_preferences.page_size` check constraint.
 */
export const PAGE_SIZES = [5, 10, 25, 50, 100] as const;

export type PageSize = (typeof PAGE_SIZES)[number];

export const DEFAULT_PAGE_SIZE: PageSize = 5;

export function isPageSize(value: number): value is PageSize {
  return (PAGE_SIZES as readonly number[]).includes(value);
}

/**
 * A size outside the list still appears, so a link from an older release keeps a
 * selected row instead of blanking the control.
 */
export function pageSizeOptions(current: number): { label: string; value: string }[] {
  const sizes = isPageSize(current) ? [...PAGE_SIZES] : [...PAGE_SIZES, current].sort((a, b) => a - b);
  return sizes.map((size) => ({ label: String(size), value: String(size) }));
}
