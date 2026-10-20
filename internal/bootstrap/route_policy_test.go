package bootstrap

import (
	"strings"
	"testing"

	"kinstack/internal/platform/authz"
)

// Every guarded workspace route must stay in the fail-closed policy table and
// resolve to a known permission (or the authenticated sentinel).
func TestRoutePolicyCoversWorkspaceRoutes(t *testing.T) {
	required := []string{
		"GET /dashboard",
		"GET /profile",
		"GET /users",
		"POST /users",
		"POST /users/invite",
		"PUT /users/:publicID",
		"POST /users/:publicID/password",
		"POST /users/:publicID/active",
		"GET /access",
		"POST /access/roles",
		"PUT /access/roles/:code",
		"POST /access/roles/:code/delete",
		"POST /access/roles/:code/permissions",
		"POST /access/users/:publicID/role",
		"POST /access/users/:publicID/permissions",
		"POST /access/users/:publicID/permissions/delete",
		"GET /audit",
		"GET /media",
		"POST /media",
		"POST /media/:publicID/delete",
		"GET /notifications",
		"GET /notifications/feed",
		"POST /notifications/read-all",
		"POST /notifications/:publicID/read",
		"GET /api-tokens",
		"POST /api-tokens",
		"POST /api-tokens/:publicID/revoke",
		"GET /passkeys",
		"POST /passkeys/register/options",
		"POST /passkeys/register",
		"POST /passkeys/:credentialID/delete",
	}

	for _, route := range required {
		policy, ok := routePolicy[route]
		if !ok {
			t.Errorf("route %q is missing from routePolicy", route)
			continue
		}
		if policy != authz.Authenticated && !authz.IsKnown(policy) {
			t.Errorf("route %q has unknown permission %q", route, policy)
		}
	}
}

func TestRoutePolicyKeysAreWellFormed(t *testing.T) {
	for route := range routePolicy {
		method, path, ok := strings.Cut(route, " ")
		if !ok || method == "" || !strings.HasPrefix(path, "/") {
			t.Errorf("malformed policy key %q", route)
		}
	}
}

// Workspace mutations must never be mounted without an explicit permission.
func TestSensitiveRoutesRequirePermission(t *testing.T) {
	sensitive := map[string]authz.Permission{
		"GET /users":  authz.PermissionUsersView,
		"POST /users": authz.PermissionUsersManage,
		"GET /access": authz.PermissionAccessManage,
		"GET /audit":  authz.PermissionAuditView,
		"GET /media":  authz.PermissionMediaView,
		"POST /media": authz.PermissionMediaManage,
	}
	for route, want := range sensitive {
		if got := routePolicy[route]; got != want {
			t.Errorf("route %q policy = %q, want %q", route, got, want)
		}
	}
}
