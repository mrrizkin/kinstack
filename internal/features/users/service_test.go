package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"kinstack/internal/platform/authz"
)

const (
	testAdminRole = "administrator"
	testStaffRole = "staff"
)

func newSeededRepository() *MemoryRepository {
	repository := NewMemoryRepository()
	repository.Seed(UserResult{
		PublicID:    "admin-1",
		Username:    "admin",
		DisplayName: "Administrator",
		Role:        testAdminRole,
		IsActive:    true,
		CreatedAt:   time.Now().UTC(),
	})
	return repository
}

func newSeededAuthorization() *authz.Service {
	repository := authz.NewMemoryRepository()
	repository.SeedRole(authz.Role{
		Code:     testAdminRole,
		Name:     "Administrator",
		IsSystem: true,
		Rank:     100,
	})
	repository.SeedRole(authz.Role{Code: testStaffRole, Name: "Staf", Rank: 10})
	repository.SeedRolePermissions(
		testAdminRole,
		authz.PermissionAccessManage,
		authz.PermissionUsersView,
		authz.PermissionUsersManage,
	)
	repository.SeedRolePermissions(testStaffRole, authz.PermissionUsersView)
	repository.SeedUser("admin-1", testAdminRole)
	return authz.New(authz.ServiceDependency{Repository: repository})
}

func newTestService() *Service {
	return NewService(ServiceDependency{
		Repository:    newSeededRepository(),
		Authorization: newSeededAuthorization(),
	})
}

func TestCreateValidatesUsername(t *testing.T) {
	service := newTestService()
	_, err := service.Create(context.Background(), CreateUserRequest{
		Username:    "a",
		DisplayName: "Staf",
		Password:    "rahasia123",
		Role:        testStaffRole,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestCreateValidatesRole(t *testing.T) {
	service := newTestService()
	_, err := service.Create(context.Background(), CreateUserRequest{
		Username:    "staf1",
		DisplayName: "Staf",
		Password:    "rahasia123",
		Role:        "role-yang-tidak-ada",
	})
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("expected invalid role, got %v", err)
	}
}

func TestCreatePersistsRole(t *testing.T) {
	service := newTestService()
	created, err := service.Create(context.Background(), CreateUserRequest{
		Username:    "staf1",
		DisplayName: "Staf Satu",
		Password:    "rahasia123",
		Role:        testStaffRole,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Role != testStaffRole || created.PublicID == "" {
		t.Fatalf("unexpected user: %#v", created)
	}
}

func TestCreateRejectsDuplicateUsername(t *testing.T) {
	service := newTestService()
	_, err := service.Create(context.Background(), CreateUserRequest{
		Username:    "admin",
		DisplayName: "Admin Lain",
		Password:    "rahasia123",
		Role:        testAdminRole,
	})
	if !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("expected username taken, got %v", err)
	}
}

func TestSetActiveProtectsLastAdministrator(t *testing.T) {
	service := newTestService()
	_, err := service.SetActive(context.Background(), "admin-1", false)
	if !errors.Is(err, ErrLastAdministrator) {
		t.Fatalf("expected last administrator protection, got %v", err)
	}
}

func TestSetActiveAllowsSecondAdministratorToBeDisabled(t *testing.T) {
	service := newTestService()
	second, err := service.Create(context.Background(), CreateUserRequest{
		Username:    "admin2",
		DisplayName: "Admin Dua",
		Password:    "rahasia123",
		Role:        testAdminRole,
	})
	if err != nil {
		t.Fatalf("create administrator: %v", err)
	}
	if _, err := service.SetActive(context.Background(), second.PublicID, false); err != nil {
		t.Fatalf("disable second administrator: %v", err)
	}
}

func TestSetActiveAllowsDisablingNonAdministrator(t *testing.T) {
	service := newTestService()
	staff, err := service.Create(context.Background(), CreateUserRequest{
		Username:    "staf1",
		DisplayName: "Staf Satu",
		Password:    "rahasia123",
		Role:        testStaffRole,
	})
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}
	if _, err := service.SetActive(context.Background(), staff.PublicID, false); err != nil {
		t.Fatalf("disable staff: %v", err)
	}
}

func TestResetPasswordRequiresMinimumLength(t *testing.T) {
	service := newTestService()
	if err := service.ResetPassword(context.Background(), "admin-1", "short"); !errors.Is(
		err,
		ErrInvalidInput,
	) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

type recordingInviter struct {
	address string
	err     error
}

func (r *recordingInviter) SendInvitation(_ context.Context, address string) error {
	r.address = address
	return r.err
}

func TestInviteCreatesInvitedAccountAndSendsLink(t *testing.T) {
	repository := newSeededRepository()
	inviter := &recordingInviter{}
	service := NewService(ServiceDependency{
		Repository:    repository,
		Authorization: newSeededAuthorization(),
		Inviter:       inviter,
	})

	invited, err := service.Invite(context.Background(), InviteUserRequest{
		Email:       "Staf.Baru@Contoh.com",
		DisplayName: "Staf Baru",
		Role:        testStaffRole,
	})
	if err != nil {
		t.Fatalf("invite: %v", err)
	}
	if !invited.Invited {
		t.Fatalf("expected invited flag, got %#v", invited)
	}
	if invited.Username != "staf.baru" || invited.Role != testStaffRole {
		t.Fatalf("unexpected invited user: %#v", invited)
	}
	if inviter.address != "staf.baru@contoh.com" {
		t.Fatalf("expected normalized invitation address, got %q", inviter.address)
	}
}

func TestInviteRejectsUnknownRole(t *testing.T) {
	service := newTestService()
	_, err := service.Invite(context.Background(), InviteUserRequest{
		Email: "staf@contoh.com",
		Role:  "role-yang-tidak-ada",
	})
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("expected invalid role, got %v", err)
	}
}
