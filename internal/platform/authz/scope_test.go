package authz

import "testing"

func TestRestrictToScopes(t *testing.T) {
	set := map[Permission]struct{}{
		PermissionUsersView:    {},
		PermissionAccessManage: {},
	}
	if got := restrictToScopes(set, nil); len(got) != 2 {
		t.Fatalf("nil scopes should not restrict, got %d", len(got))
	}
	if got := restrictToScopes(set, []string{"users.view"}); len(got) != 1 {
		t.Fatalf("expected a single scoped permission, got %d", len(got))
	}
	if got := restrictToScopes(set, []string{}); len(got) != 0 {
		t.Fatalf("empty scopes should deny everything, got %d", len(got))
	}
	if got := restrictToScopes(set, []string{"unknown.permission"}); len(got) != 0 {
		t.Fatalf("unknown scopes should deny everything, got %d", len(got))
	}
}
