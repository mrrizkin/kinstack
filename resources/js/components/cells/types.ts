/**
 * Structural types shared by table cell components.
 *
 * Cells never import feature types from `resources/js/pages`; callers map a row
 * onto these shapes at the column boundary.
 */
export type CellBadgeColor = "primary" | "secondary" | "success" | "info" | "warning" | "error" | "neutral";

export type CellBadgeVariant = "solid" | "outline" | "soft" | "subtle";

export type CellBadgeSize = "sm" | "md" | "lg";

export interface CellBadgeSpec {
  label: string;
  color?: CellBadgeColor;
}
