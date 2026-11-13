/**
 * Cell prop types, so cells never import feature types from `pages`.
 */
export type CellBadgeColor = "primary" | "secondary" | "success" | "info" | "warning" | "error" | "neutral";

export type CellBadgeVariant = "solid" | "outline" | "soft" | "subtle";

export type CellBadgeSize = "sm" | "md" | "lg";

export interface CellBadgeSpec {
  label: string;
  color?: CellBadgeColor;
}
