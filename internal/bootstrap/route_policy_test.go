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
		if policy != authz.Authenticated && policy != authz.BrowserSession && !authz.IsKnown(policy) {
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

// Credential and session management must be browser-session only: the
// Authenticated sentinel skips scope restriction, so an API token would
// otherwise be able to rotate the credentials it authenticates with.
func TestCredentialRoutesRequireBrowserSession(t *testing.T) {
	credentialRoutes := []string{
		"GET /profile",
		"POST /profile",
		"POST /profile/password",
		"POST /profile/sessions/:publicID/revoke",
		"POST /profile/sessions/revoke-others",
		"POST /profile/two-factor/begin",
		"POST /profile/two-factor/confirm",
		"POST /profile/two-factor/disable",
		"POST /profile/two-factor/recovery-codes",
		"GET /api-tokens",
		"POST /api-tokens",
		"POST /api-tokens/:publicID/revoke",
		"GET /passkeys",
		"POST /passkeys/register/options",
		"POST /passkeys/register",
		"POST /passkeys/:credentialID/delete",
	}
	for _, route := range credentialRoutes {
		if got := routePolicy[route]; got != authz.BrowserSession {
			t.Errorf("credential route %q policy = %q, want %q", route, got, authz.BrowserSession)
		}
	}
}
