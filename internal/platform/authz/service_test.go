package authz

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"
)

const adminRole = "administrator"

func newTestService(t *testing.T) (*Service, *MemoryRepository) {
	t.Helper()
	repository := NewMemoryRepository()
	repository.SeedRole(Role{Code: adminRole, Name: "Administrator", IsSystem: true, Rank: 100})
	repository.SeedRole(Role{Code: "staff", Name: "Staf", IsSystem: true, Rank: 10})
	repository.SeedRolePermissions(adminRole, All()...)
	repository.SeedRolePermissions(
		"staff",
		PermissionUsersView,
	)
	repository.SeedUser("administrator-1", adminRole)
	repository.SeedUser("staff-1", "staff")
	return New(ServiceDependency{Repository: repository}), repository
}

// adminContext asserts the seeded administrator as the acting principal.
func adminContext() context.Context {
	return WithPrincipal(
		context.Background(),
		Principal{PublicID: "administrator-1", Role: adminRole},
	)
}

func TestRoleWithAllPermissions(t *testing.T) {
	service, _ := newTestService(t)
	principal := Principal{PublicID: "administrator-1", Role: adminRole}
	for _, permission := range All() {
		if decision := service.Can(
			context.Background(),
			principal,
			permission,
			nil,
		); !decision.Allowed {
			t.Fatalf("administrator denied %s: %+v", permission, decision)
		}
	}
}

func TestSuperuserBypassesRBAC(t *testing.T) {
	service := New(ServiceDependency{Repository: NewMemoryRepository()})
	principal := Principal{PublicID: "break-glass", Superuser: true}

	for _, permission := range All() {
		if decision := service.Can(
			context.Background(),
			principal,
			permission,
			nil,
		); !decision.Allowed {
			t.Fatalf("superuser denied %s: %+v", permission, decision)
		}
	}
	permissions, err := service.Permissions(context.Background(), principal)
	if err != nil {
		t.Fatalf("permissions: %v", err)
	}
	if len(permissions) != len(All()) {
		t.Fatalf("expected every known permission, got %#v", permissions)
	}
	if has, err := service.Has(
		context.Background(),
		principal,
		PermissionAccessManage,
	); err != nil ||
		!has {
		t.Fatalf("superuser should hold access.manage: has=%v err=%v", has, err)
	}
}

func TestRoleRBACBaseline(t *testing.T) {
	service, _ := newTestService(t)
	principal := Principal{PublicID: "staff-1", Role: "staff"}

	if decision := service.Can(
		context.Background(),
		principal,
		PermissionUsersView,
		nil,
	); !decision.Allowed {
		t.Fatalf("staff denied users.view: %+v", decision)
	}
	if decision := service.Can(
		context.Background(),
		principal,
		PermissionAccessManage,
		nil,
	); decision.Allowed {
		t.Fatalf("staff unexpectedly allowed access.manage")
	}
}

func TestAllowOverrideGrantsPermission(t *testing.T) {
	service, repository := newTestService(t)
	repository.SeedUser("staff-1", "staff")
	principal := Principal{PublicID: "staff-1", Role: "staff"}

	if _, err := service.SetOverride(
		adminContext(),
		"staff-1",
		PermissionAccessManage,
		EffectAllow,
		"boleh kelola akses",
		nil,
	); err != nil {
		t.Fatalf("set override: %v", err)
	}

	decision := service.Can(context.Background(), principal, PermissionAccessManage, nil)
	if !decision.Allowed {
		t.Fatalf("allow override did not grant permission: %+v", decision)
	}
}

func TestDenyOverrideWinsOverRole(t *testing.T) {
	service, repository := newTestService(t)
	repository.SeedUser("staff-1", "staff")
	principal := Principal{PublicID: "staff-1", Role: "staff"}

	if _, err := service.SetOverride(
		adminContext(),
		"staff-1",
		PermissionUsersView,
		EffectDeny,
		"tidak boleh lihat pengguna",
		nil,
	); err != nil {
		t.Fatalf("set override: %v", err)
	}

	decision := service.Can(context.Background(), principal, PermissionUsersView, nil)
	if decision.Allowed {
		t.Fatalf("deny override did not revoke permission: %+v", decision)
	}
}

func TestExpiredOverrideIsIgnored(t *testing.T) {
	repository := NewMemoryRepository()
	repository.SeedRole(Role{Code: "staff", Name: "Kasir"})
	repository.SeedRolePermissions("staff", PermissionUsersView)
	repository.SeedUser("staff-1", "staff")

	now := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	service := New(ServiceDependency{Repository: repository, Now: func() time.Time { return now }})

	expired := now.Add(-time.Hour)
	if _, err := service.SetOverride(
		superuserContext(),
		"staff-1",
		PermissionAccessManage,
		EffectAllow,
		"kedaluwarsa",
		&expired,
	); err != nil {
		t.Fatalf("set override: %v", err)
	}

	decision := service.Can(
		context.Background(),
		Principal{PublicID: "staff-1", Role: "staff"},
		PermissionAccessManage,
		nil,
	)
	if decision.Allowed {
		t.Fatalf("expired override still granted permission: %+v", decision)
	}
}

// policyTestService wires a custom ABAC engine so the generic policy types can
// be exercised against a base permission without depending on a feature slice.
func policyTestService(t *testing.T, policies ...Policy) (*Service, *MemoryRepository) {
	t.Helper()
	repository := NewMemoryRepository()
	repository.SeedRole(Role{Code: adminRole, Name: "Administrator", IsSystem: true, Rank: 100})
	repository.SeedRole(Role{Code: "staff", Name: "Staf", IsSystem: true, Rank: 10})
	repository.SeedRolePermissions(adminRole, PermissionUsersManage, PermissionAccessManage)
	repository.SeedRolePermissions("staff", PermissionUsersManage, PermissionAccessManage)
	repository.SeedUser("administrator-1", adminRole)
	repository.SeedUser("staff-1", "staff")
	service := New(ServiceDependency{Repository: repository, Engine: NewEngine(policies...)})
	return service, repository
}

func TestOwnershipPolicyDeniesOtherUser(t *testing.T) {
	service, _ := policyTestService(t, OwnershipPolicy{
		PolicyName: "test.ownership",
		Permission: PermissionUsersManage,
		ExemptRole: adminRole,
		Message:    "hanya boleh mengubah data sendiri",
	})
	cashier := Principal{PublicID: "staff-1", Role: "staff"}

	denied := service.Can(context.Background(), cashier, PermissionUsersManage, Resource{
		ResourceOwner: "staff-2",
	})
	if denied.Allowed {
		t.Fatalf("ownership policy should deny another user's record")
	}

	if decision := service.Can(context.Background(), cashier, PermissionUsersManage, Resource{
		ResourceOwner: "staff-1",
	}); !decision.Allowed {
		t.Fatalf("ownership policy should allow own record: %+v", decision)
	}

	owner := Principal{PublicID: "administrator-1", Role: "administrator"}
	if decision := service.Can(context.Background(), owner, PermissionUsersManage, Resource{
		ResourceOwner: "staff-2",
	}); !decision.Allowed {
		t.Fatalf("owner should bypass ownership policy: %+v", decision)
	}
}

func TestAmountLimitPolicyDeniesRole(t *testing.T) {
	service, _ := policyTestService(t, AmountLimitPolicy{
		PolicyName: "test.amount",
		Permission: PermissionAccessManage,
		ExemptRole: adminRole,
		Limit:      5_000_000,
		Message:    "nominal di atas batas memerlukan pemilik",
	})
	cashier := Principal{PublicID: "staff-1", Role: "staff"}
	owner := Principal{PublicID: "administrator-1", Role: "administrator"}

	if decision := service.Can(context.Background(), cashier, PermissionAccessManage, Resource{
		ResourceAmount: 1_000_000,
	}); !decision.Allowed {
		t.Fatalf("amount under the limit should be allowed: %+v", decision)
	}

	if decision := service.Can(context.Background(), cashier, PermissionAccessManage, Resource{
		ResourceAmount: 50_000_000,
	}); decision.Allowed {
		t.Fatalf("amount over the limit should require owner: %+v", decision)
	}

	if decision := service.Can(context.Background(), owner, PermissionAccessManage, Resource{
		ResourceAmount: 50_000_000,
	}); !decision.Allowed {
		t.Fatalf("owner should bypass amount limit: %+v", decision)
	}
}

func TestUnknownPermissionIsDenied(t *testing.T) {
	service, _ := newTestService(t)
	owner := Principal{PublicID: "administrator-1", Role: "administrator"}
	if decision := service.Can(
		context.Background(),
		owner,
		Permission("nope.nope"),
		nil,
	); decision.Allowed {
		t.Fatalf("unknown permission must be denied")
	}
}

func TestRoleMutationValidation(t *testing.T) {
	service, _ := newTestService(t)
	ctx := superuserContext()

	if _, err := service.CreateRole(ctx, "Auditor", "Auditor", "", 20); err != nil {
		t.Fatalf("create role: %v", err)
	}
	if _, err := service.CreateRole(ctx, "x", "Terlalu pendek", "", 20); err == nil {
		t.Fatalf("expected invalid role code to fail")
	}
	if err := service.ReplaceRolePermissions(
		ctx,
		"auditor",
		[]Permission{PermissionAccessManage, "nope"},
	); err == nil {
		t.Fatalf("expected unknown permission to fail")
	}
	if _, err := service.UpdateRole(
		ctx,
		"staff",
		"Kasir Depan",
		"Kasir shift depan",
		15,
	); !errors.Is(
		err,
		ErrSystemRole,
	) {
		t.Fatalf("system role metadata must be immutable, got %v", err)
	}
	if err := service.DeleteRole(ctx, "staff"); err != ErrSystemRole {
		t.Fatalf("expected system role delete protection, got %v", err)
	}
}

func TestMigrationSeedUsesKnownPermissions(t *testing.T) {
	contents, err := os.ReadFile(
		"../../../db/migrations/202601010005_insert_authorization_seed.sql",
	)
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	insert := regexp.MustCompile(`(?s)INSERT INTO role_permissions.*?;`)
	row := regexp.MustCompile(`\('[^']+',\s*'([^']+)'\)`)
	matches := row.FindAllStringSubmatch(string(insert.Find(contents)), -1)
	if len(matches) == 0 {
		t.Fatalf("no seeded role permissions found in migration")
	}
	for _, match := range matches {
		if !IsKnown(Permission(match[1])) {
			t.Errorf("migration seeds unknown permission %q", match[1])
		}
	}
}

func TestUserOverridesManyGroupsByUser(t *testing.T) {
	service, repository := newTestService(t)
	if _, err := repository.SetUserOverride(context.Background(), SetOverrideInput{
		UserPublicID: "staff-1",
		Permission:   PermissionUsersManage,
		Effect:       EffectAllow,
		Reason:       "temporary cover",
	}); err != nil {
		t.Fatalf("seed override: %v", err)
	}

	grouped, err := service.UserOverridesMany(
		context.Background(),
		[]string{"staff-1", "administrator-1"},
	)
	if err != nil {
		t.Fatalf("batch overrides: %v", err)
	}
	if got := grouped["staff-1"]; len(got) != 1 || got[0].Permission != PermissionUsersManage {
		t.Fatalf("unexpected staff overrides: %+v", got)
	}
	empty, ok := grouped["administrator-1"]
	if !ok {
		t.Fatal("every requested user must be present in the result map")
	}
	if empty == nil || len(empty) != 0 {
		t.Fatalf("expected an empty slice for a user without overrides, got %#v", empty)
	}

	none, err := service.UserOverridesMany(context.Background(), nil)
	if err != nil {
		t.Fatalf("empty batch: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected an empty result, got %#v", none)
	}
}

func TestEffectiveBoundsTheResolutionCache(t *testing.T) {
	service, _ := newTestService(t)
	service.ttl = time.Minute

	service.mu.Lock()
	for index := 0; index < maxCacheEntries; index++ {
		service.cache[fmt.Sprintf("cached-%d", index)] = resolvedPermissions{
			role:      adminRole,
			set:       map[Permission]struct{}{PermissionUsersView: {}},
			expiresAt: service.now().Add(time.Minute),
		}
	}
	service.mu.Unlock()

	if _, err := service.Permissions(
		context.Background(),
		Principal{PublicID: "fresh-1", Role: adminRole},
	); err != nil {
		t.Fatalf("resolve permissions: %v", err)
	}

	service.mu.RLock()
	size := len(service.cache)
	_, cached := service.cache["fresh-1"]
	service.mu.RUnlock()
	if size >= maxCacheEntries {
		t.Fatalf("cache must stay bounded, got %d entries", size)
	}
	if !cached {
		t.Fatal("the new resolution must be cached after eviction")
	}
}

func TestRolePermissionsManyFiltersUnknownCodes(t *testing.T) {
	service, repository := newTestService(t)
	repository.SeedRolePermissions("staff", PermissionUsersView, Permission("made.up"))

	grouped, err := service.RolePermissionsMany(
		context.Background(),
		[]string{"staff", adminRole, "missing"},
	)
	if err != nil {
		t.Fatalf("batch permissions: %v", err)
	}
	if got := grouped["staff"]; len(got) != 1 || got[0] != PermissionUsersView {
		t.Fatalf("unknown codes must be filtered, got %+v", got)
	}
	if len(grouped[adminRole]) == 0 {
		t.Fatal("administrator permissions must be returned")
	}
	empty, ok := grouped["missing"]
	if !ok || empty == nil || len(empty) != 0 {
		t.Fatalf("missing roles must map to an empty slice, got %#v", empty)
	}
}
