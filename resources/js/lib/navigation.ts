/**
 * Shared visibility rules for sidebar and dashboard entries. Feature flags come
 * from the server so the UI never links to a route that was not mounted.
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
  if (item.requiresMedia && !mediaEnabled) return false;
  return !item.permission || can(item.permission);
}
