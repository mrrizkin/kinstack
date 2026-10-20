package authz

import (
	"context"
	"errors"
	"testing"
)

// newAuthorityService seeds a hierarchy used by the delegation tests:
//
//	administrator (100, every permission, system)
//	access-admin  (90,  access.manage + users.view)
//	junior-admin  (50,  access.manage + every permission)
//	staff         (10,  users.view)
func newAuthorityService(t *testing.T) (*Service, *MemoryRepository) {
	t.Helper()
	repository := NewMemoryRepository()
	repository.SeedRole(Role{Code: "administrator", Name: "Administrator", IsSystem: true, Rank: 100})
	repository.SeedRole(Role{Code: "access-admin", Name: "Access Admin", Rank: 90})
	repository.SeedRole(Role{Code: "junior-admin", Name: "Junior Admin", Rank: 50})
	repository.SeedRole(Role{Code: "staff", Name: "Staf", IsSystem: true, Rank: 10})
	repository.SeedRolePermissions("administrator", All()...)
	repository.SeedRolePermissions(
		"access-admin",
		PermissionAccessManage,
		PermissionUsersView,
	)
	repository.SeedRolePermissions("junior-admin", All()...)
	repository.SeedRolePermissions("staff", PermissionUsersView)
	repository.SeedUser("administrator-1", "administrator")
	repository.SeedUser("access-admin-1", "access-admin")
	repository.SeedUser("junior-admin-1", "junior-admin")
	repository.SeedUser("staff-1", "staff")
	return New(ServiceDependency{Repository: repository}), repository
}

func superuserContext() context.Context {
	return WithPrincipal(context.Background(), Principal{PublicID: "break-glass", Superuser: true})
}

func actorContext(publicID, role string) context.Context {
	return WithPrincipal(context.Background(), Principal{PublicID: publicID, Role: role})
}

func TestAdministratorCanGrantAnyRole(t *testing.T) {
	service, _ := newAuthorityService(t)
	ctx := actorContext("administrator-1", "administrator")

	for _, role := range []string{"administrator", "access-admin", "staff"} {
		if err := service.AuthorizeRoleGrant(ctx, role); err != nil {
			t.Fatalf("administrator should grant %s: %v", role, err)
		}
	}
	if err := service.AssignRole(ctx, "staff-1", "staff"); err != nil {
		t.Fatalf("assign role: %v", err)
	}
}

func TestGrantIsLimitedToTheActorsPermissions(t *testing.T) {
	service, _ := newAuthorityService(t)
	ctx := actorContext("access-admin-1", "access-admin")

	// access-admin holds access.manage but not users.manage, which the
	// administrator role carries, so handing it out would be an escalation.
	if err := service.AuthorizeRoleGrant(ctx, "administrator"); !errors.Is(
		err,
		ErrPrivilegeEscalation,
	) {
		t.Fatalf("expected a subset violation, got %v", err)
	}
	if err := service.AssignRole(ctx, "staff-1", "administrator"); !errors.Is(
		err,
		ErrPrivilegeEscalation,
	) {
		t.Fatalf("expected AssignRole to deny the escalation, got %v", err)
	}

	// A role the actor can fully cover is allowed.
	if err := service.AuthorizeRoleGrant(ctx, "staff"); err != nil {
		t.Fatalf("expected staff role to be grantable: %v", err)
	}
}

func TestGrantIsLimitedToTheActorsRank(t *testing.T) {
	service, _ := newAuthorityService(t)
	ctx := actorContext("junior-admin-1", "junior-admin")

	// junior-admin holds every permission but ranks below administrator.
	if err := service.AuthorizeRoleGrant(ctx, "administrator"); !errors.Is(
		err,
		ErrPrivilegeEscalation,
	) {
		t.Fatalf("expected a rank violation, got %v", err)
	}
	if err := service.AuthorizeUserChange(ctx, "administrator-1"); !errors.Is(
		err,
		ErrPrivilegeEscalation,
	) {
		t.Fatalf("expected a target rank violation, got %v", err)
	}
	if err := service.AuthorizeUserChange(ctx, "staff-1"); err != nil {
		t.Fatalf("lower-ranked target should be manageable: %v", err)
	}
}

func TestSelfChangeIsRejected(t *testing.T) {
	service, _ := newAuthorityService(t)
	ctx := actorContext("administrator-1", "administrator")

	if err := service.AuthorizeUserChange(ctx, "administrator-1"); !errors.Is(
		err,
		ErrSelfChange,
	) {
		t.Fatalf("expected ErrSelfChange, got %v", err)
	}
	if err := service.AssignRole(ctx, "administrator-1", "staff"); !errors.Is(
		err,
		ErrSelfChange,
	) {
		t.Fatalf("expected AssignRole to reject self-demotion, got %v", err)
	}
}

func TestTokenActorsCannotManageAuthority(t *testing.T) {
	service, _ := newAuthorityService(t)
	ctx := WithPrincipal(context.Background(), Principal{
		PublicID: "administrator-1",
		Role:     "administrator",
		Scopes:   []string{string(PermissionAccessManage)},
	})

	if err := service.AuthorizeRoleGrant(ctx, "staff"); !errors.Is(
		err,
		ErrManagementSession,
	) {
		t.Fatalf("expected a token denial, got %v", err)
	}
}

func TestMissingPrincipalFailsClosed(t *testing.T) {
	service, _ := newAuthorityService(t)
	if err := service.AuthorizeRoleGrant(context.Background(), "staff"); err == nil {
		t.Fatal("expected a missing principal to be denied")
	}
}

func TestSystemRolesAreImmutable(t *testing.T) {
	service, _ := newAuthorityService(t)
	ctx := superuserContext()

	if err := service.AuthorizeRoleEdit(ctx, "staff", 10); !errors.Is(err, ErrSystemRole) {
		t.Fatalf("expected ErrSystemRole, got %v", err)
	}
	if _, err := service.UpdateRole(ctx, "staff", "Baru", "", 10); !errors.Is(
		err,
		ErrSystemRole,
	) {
		t.Fatalf("expected UpdateRole to reject a system role, got %v", err)
	}
	if err := service.ReplaceRolePermissions(
		ctx,
		"staff",
		[]Permission{PermissionUsersView},
	); !errors.Is(err, ErrSystemRole) {
		t.Fatalf("expected ReplaceRolePermissions to reject a system role, got %v", err)
	}
	if err := service.DeleteRole(ctx, "staff"); !errors.Is(err, ErrSystemRole) {
		t.Fatalf("expected DeleteRole to reject a system role, got %v", err)
	}
}

func TestRoleEditCannotExceedActorRank(t *testing.T) {
	service, _ := newAuthorityService(t)
	ctx := actorContext("junior-admin-1", "junior-admin")

	if err := service.AuthorizeRoleEdit(ctx, "access-admin", 90); !errors.Is(
		err,
		ErrPrivilegeEscalation,
	) {
		t.Fatalf("expected a rank violation, got %v", err)
	}
	if err := service.AuthorizeNewRole(ctx, 80); !errors.Is(err, ErrPrivilegeEscalation) {
		t.Fatalf("expected a new-role rank violation, got %v", err)
	}
	if err := service.AuthorizeNewRole(ctx, 20); err != nil {
		t.Fatalf("lower rank should be allowed: %v", err)
	}
}

func TestOverrideRequiresTheGrantedPermission(t *testing.T) {
	service, _ := newAuthorityService(t)
	ctx := actorContext("access-admin-1", "access-admin")

	if _, err := service.SetOverride(
		ctx,
		"staff-1",
		PermissionUsersManage,
		EffectAllow,
		"butuh kelola pengguna",
		nil,
	); !errors.Is(err, ErrPrivilegeEscalation) {
		t.Fatalf("expected an override escalation denial, got %v", err)
	}
	if _, err := service.SetOverride(
		ctx,
		"staff-1",
		PermissionUsersView,
		EffectAllow,
		"boleh lihat",
		nil,
	); err != nil {
		t.Fatalf("expected a covered permission to be grantable: %v", err)
	}
}
