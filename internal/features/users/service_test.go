package users

import (
	"context"
	"errors"
	"testing"
	"time"

	"kinstack/internal/pkg/pagination"
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

func TestParsePageQueryRejectsInvalidValues(t *testing.T) {
	cases := []struct{ name, page, perPage string }{
		{name: "non-numeric page", page: "x"},
		{name: "zero page", page: "0"},
		{name: "negative page", page: "-1"},
		{name: "non-numeric per page", perPage: "x"},
		{name: "zero per page", perPage: "0"},
		{name: "too many per page", perPage: "101"},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParsePageQuery(test.page, test.perPage); !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestParsePageQueryAppliesDefaults(t *testing.T) {
	query, err := ParsePageQuery("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query.Page != 1 || query.PerPage != DefaultPerPage {
		t.Fatalf("unexpected query: %+v", query)
	}

	query, err = ParsePageQuery("3", "100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if query.Page != 3 || query.PerPage != MaxPerPage {
		t.Fatalf("unexpected query: %+v", query)
	}
}

func TestListPaginatesAndCounts(t *testing.T) {
	repository := newSeededRepository()
	base := time.Now().UTC().Add(-time.Hour)
	for index := 0; index < 4; index++ {
		repository.Seed(UserResult{
			PublicID:    "staff-" + string(rune('a'+index)),
			Username:    "staff" + string(rune('a'+index)),
			DisplayName: "Staf",
			Role:        testStaffRole,
			IsActive:    index%2 == 0,
			CreatedAt:   base.Add(time.Duration(index) * time.Minute),
		})
	}
	service := NewService(ServiceDependency{
		Repository:    repository,
		Authorization: newSeededAuthorization(),
	})

	list, err := service.List(context.Background(), pagination.Query{Page: 2, PerPage: 2})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(list.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(list.Users))
	}
	if list.Page.Total != 5 || list.Page.TotalPages != 3 || list.Page.From != 3 || list.Page.To != 4 {
		t.Fatalf("unexpected page: %+v", list.Page)
	}

	counts, err := service.Count(context.Background())
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if counts.Total != 5 || counts.Active != 3 {
		t.Fatalf("unexpected counts: %+v", counts)
	}
}
