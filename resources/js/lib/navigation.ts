/**
 * Sidebar and dashboard visibility rules. Flags come from the server, so a link
 * never points at an unmounted route.
 */
export interface NavigationGate {
  permission?: string;
  requiresMedia?: boolean;
}

export function isNavigationItemVisible(
  item: NavigationGate,
  can: (code: string) => boolean,
  mediaEnabled: boolean,
): boolean {
  if (item.requiresMedia && !mediaEnabled) {
    return false;
  }
  return !item.permission || can(item.permission);
}
