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

func newSeededAuthorizationRepository() *authz.MemoryRepository {
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
	return repository
}

func newSeededAuthorization() *authz.Service {
	return authz.New(authz.ServiceDependency{Repository: newSeededAuthorizationRepository()})
}

// testContext asserts the seeded administrator as the acting principal so
// authorization mutations can enforce delegation rules.
func testContext() context.Context {
	return authz.WithPrincipal(context.Background(), authz.Principal{
		PublicID: "admin-1",
		Role:     testAdminRole,
	})
}

func newTestService() *Service {
	return NewService(ServiceDependency{
		Repository:    newSeededRepository(),
		Authorization: newSeededAuthorization(),
	})
}

func TestCreateValidatesUsername(t *testing.T) {
	service := newTestService()
	_, err := service.Create(testContext(), CreateUserRequest{
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
	_, err := service.Create(testContext(), CreateUserRequest{
		Username:    "staf1",
		DisplayName: "Staf",
		Password:    "rahasia123",
		Role:        "missing-role",
	})
	if !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("expected invalid role, got %v", err)
	}
}

func TestCreatePersistsRole(t *testing.T) {
	service := newTestService()
	created, err := service.Create(testContext(), CreateUserRequest{
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
	_, err := service.Create(testContext(), CreateUserRequest{
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
	_, err := service.SetActive(testContext(), "admin-1", false)
	if !errors.Is(err, ErrLastAdministrator) {
		t.Fatalf("expected last administrator protection, got %v", err)
	}
}

func TestSetActiveAllowsSecondAdministratorToBeDisabled(t *testing.T) {
	service := newTestService()
	second, err := service.Create(testContext(), CreateUserRequest{
		Username:    "admin2",
		DisplayName: "Admin Dua",
		Password:    "rahasia123",
		Role:        testAdminRole,
	})
	if err != nil {
		t.Fatalf("create administrator: %v", err)
	}
	if _, err := service.SetActive(testContext(), second.PublicID, false); err != nil {
		t.Fatalf("disable second administrator: %v", err)
	}
}

func TestSetActiveAllowsDisablingNonAdministrator(t *testing.T) {
	service := newTestService()
	staff, err := service.Create(testContext(), CreateUserRequest{
		Username:    "staf1",
		DisplayName: "Staf Satu",
		Password:    "rahasia123",
		Role:        testStaffRole,
	})
	if err != nil {
		t.Fatalf("create staff: %v", err)
	}
	if _, err := service.SetActive(testContext(), staff.PublicID, false); err != nil {
		t.Fatalf("disable staff: %v", err)
	}
}

func TestResetPasswordRequiresMinimumLength(t *testing.T) {
	service := newTestService()
	if err := service.ResetPassword(testContext(), "admin-1", "short"); !errors.Is(
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

	invited, err := service.Invite(testContext(), InviteUserRequest{
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
	_, err := service.Invite(testContext(), InviteUserRequest{
		Email: "staf@contoh.com",
		Role:  "missing-role",
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

	list, err := service.List(testContext(), pagination.Query{Page: 2, PerPage: 2})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if len(list.Users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(list.Users))
	}
	if list.Page.Total != 5 || list.Page.TotalPages != 3 || list.Page.From != 3 ||
		list.Page.To != 4 {
		t.Fatalf("unexpected page: %+v", list.Page)
	}

	past, err := service.List(testContext(), pagination.Query{Page: 99, PerPage: 2})
	if err != nil {
		t.Fatalf("list past the end: %v", err)
	}
	if past.Page.Page != 3 || len(past.Users) != 1 {
		t.Fatalf(
			"past-end page must clamp to the last page: page=%+v rows=%d",
			past.Page,
			len(past.Users),
		)
	}

	counts, err := service.Count(testContext())
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if counts.Total != 5 || counts.Active != 3 {
		t.Fatalf("unexpected counts: %+v", counts)
	}
}

type recordingRevoker struct {
	revoked []string
}

func (r *recordingRevoker) RevokeAll(_ context.Context, userPublicID string) error {
	r.revoked = append(r.revoked, userPublicID)
	return nil
}

func TestAdminResetPasswordRevokesCredentials(t *testing.T) {
	revoker := &recordingRevoker{}
	service := NewService(ServiceDependency{
		Repository:    newSeededRepository(),
		Authorization: newSeededAuthorization(),
		Revoker:       revoker,
	})
	if err := service.ResetPassword(testContext(), "admin-1", "password-baru-1"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if len(revoker.revoked) != 1 || revoker.revoked[0] != "admin-1" {
		t.Fatalf("expected credentials to be revoked, got %#v", revoker.revoked)
	}
}

func TestAccessViewPaginatesUsersAndBatchesOverrides(t *testing.T) {
	repository := newSeededRepository()
	base := time.Now().UTC().Add(-time.Hour)
	for index := 0; index < 4; index++ {
		id := "staff-" + string(rune('a'+index))
		repository.Seed(UserResult{
			PublicID:    id,
			Username:    "staff" + string(rune('a'+index)),
			DisplayName: "Staf",
			Role:        testStaffRole,
			IsActive:    true,
			CreatedAt:   base.Add(time.Duration(index) * time.Minute),
		})
	}
	authorizationRepository := newSeededAuthorizationRepository()
	authorizationRepository.SeedUser("staff-a", testStaffRole)
	if _, err := authorizationRepository.SetUserOverride(
		context.Background(),
		authz.SetOverrideInput{
			UserPublicID: "staff-a",
			Permission:   authz.PermissionUsersView,
			Effect:       authz.EffectAllow,
			Reason:       "onboarding",
		},
	); err != nil {
		t.Fatalf("seed override: %v", err)
	}
	service := NewService(ServiceDependency{
		Repository:    repository,
		Authorization: authz.New(authz.ServiceDependency{Repository: authorizationRepository}),
	})

	props, err := service.AccessView(testContext(), pagination.Query{Page: 1, PerPage: 2})
	if err != nil {
		t.Fatalf("access view: %v", err)
	}
	if len(props.Users) != 2 {
		t.Fatalf("expected 2 users on the page, got %d", len(props.Users))
	}
	if props.Page.Total != 5 || props.Page.TotalPages != 3 || props.Page.From != 1 ||
		props.Page.To != 2 {
		t.Fatalf("unexpected page: %+v", props.Page)
	}
	if len(props.Roles) != 2 || len(props.Permissions) == 0 {
		t.Fatalf("access view must include roles and permissions: %d roles, %d permissions",
			len(props.Roles), len(props.Permissions))
	}
	rolePermissions := 0
	for _, role := range props.Roles {
		if role.Code == testAdminRole && len(role.Permissions) == 0 {
			t.Fatal("administrator role must carry its permission set")
		}
		rolePermissions += len(role.Permissions)
	}
	if rolePermissions == 0 {
		t.Fatal("role permission sets must be populated from the batched query")
	}

	found := false
	for _, user := range props.Users {
		if user.PublicID != "staff-a" {
			if len(user.Overrides) != 0 {
				t.Fatalf("user %s must not carry another user's overrides", user.PublicID)
			}
			continue
		}
		found = true
		if len(user.Overrides) != 1 ||
			user.Overrides[0].Permission != string(authz.PermissionUsersView) {
			t.Fatalf("expected the staff-a override, got %+v", user.Overrides)
		}
	}
	if !found {
		t.Fatal("expected staff-a on the first page")
	}
}
