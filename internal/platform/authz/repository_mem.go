package authz

import (
	"context"
	"sort"
	"sync"
	"time"

	"kinstack/internal/pkg/pagination"
)

// MemoryRepository is an in-memory Repository used by tests and local wiring.
type MemoryRepository struct {
	mu        sync.Mutex
	roles     map[string]Role
	rolePerms map[string]map[Permission]struct{}
	overrides map[string]map[Permission]Override
	userRoles map[string]string
	now       func() time.Time
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		roles:     make(map[string]Role),
		rolePerms: make(map[string]map[Permission]struct{}),
		overrides: make(map[string]map[Permission]Override),
		userRoles: make(map[string]string),
		now:       time.Now,
	}
}

// SeedRole inserts or replaces a role.
func (r *MemoryRepository) SeedRole(role Role) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if role.CreatedAt.IsZero() {
		role.CreatedAt = r.now()
	}
	if role.UpdatedAt.IsZero() {
		role.UpdatedAt = role.CreatedAt
	}
	r.roles[role.Code] = role
}

// SeedRolePermissions replaces a role's permission set.
func (r *MemoryRepository) SeedRolePermissions(code string, permissions ...Permission) {
	r.mu.Lock()
	defer r.mu.Unlock()
	set := make(map[Permission]struct{}, len(permissions))
	for _, permission := range permissions {
		set[permission] = struct{}{}
	}
	r.rolePerms[code] = set
}

// SeedUser registers a user's role.
func (r *MemoryRepository) SeedUser(publicID, role string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userRoles[publicID] = role
}

func (r *MemoryRepository) ListRoles(_ context.Context, query pagination.Query) ([]Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	roles := make([]Role, 0, len(r.roles))
	for _, role := range r.roles {
		roles = append(roles, role)
	}
	sort.Slice(roles, func(i, j int) bool {
		if roles[i].Rank != roles[j].Rank {
			return roles[i].Rank > roles[j].Rank
		}
		return roles[i].Code < roles[j].Code
	})
	start, end := pagination.Window(query, len(roles))
	return roles[start:end], nil
}

func (r *MemoryRepository) RoleNamesByCode(
	_ context.Context,
	codes []string,
) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	names := make(map[string]string, len(codes))
	for _, code := range codes {
		if role, ok := r.roles[code]; ok {
			names[code] = role.Name
		}
	}
	return names, nil
}

func (r *MemoryRepository) CountRoles(context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.roles), nil
}

// FindRole resolves a role.
func (r *MemoryRepository) FindRole(_ context.Context, code string) (Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	role, ok := r.roles[code]
	if !ok {
		return Role{}, ErrRoleNotFound
	}
	return role, nil
}

func (r *MemoryRepository) CreateRole(_ context.Context, input CreateRoleInput) (Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.roles[input.Code]; exists {
		return Role{}, ErrRoleCodeTaken
	}
	role := Role{
		Code:        input.Code,
		Name:        input.Name,
		Description: input.Description,
		Rank:        input.Rank,
		CreatedAt:   r.now(),
		UpdatedAt:   r.now(),
	}
	r.roles[input.Code] = role
	return role, nil
}

// UpdateRole edits a role.
func (r *MemoryRepository) UpdateRole(_ context.Context, input UpdateRoleInput) (Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	role, ok := r.roles[input.Code]
	if !ok {
		return Role{}, ErrRoleNotFound
	}
	role.Name = input.Name
	role.Description = input.Description
	role.Rank = input.Rank
	role.UpdatedAt = r.now()
	r.roles[input.Code] = role
	return role, nil
}

func (r *MemoryRepository) DeleteRole(_ context.Context, code string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.roles[code]; !ok {
		return ErrRoleNotFound
	}
	delete(r.roles, code)
	delete(r.rolePerms, code)
	return nil
}

func (r *MemoryRepository) RolePermissions(_ context.Context, code string) ([]Permission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	set := r.rolePerms[code]
	permissions := make([]Permission, 0, len(set))
	for permission := range set {
		permissions = append(permissions, permission)
	}
	sort.Slice(permissions, func(i, j int) bool { return permissions[i] < permissions[j] })
	return permissions, nil
}

// RolePermissionsMany returns the permission sets of several roles at once.
func (r *MemoryRepository) RolePermissionsMany(
	_ context.Context,
	codes []string,
) (map[string][]Permission, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make(map[string][]Permission, len(codes))
	for _, code := range codes {
		set := r.rolePerms[code]
		permissions := make([]Permission, 0, len(set))
		for permission := range set {
			permissions = append(permissions, permission)
		}
		sort.Slice(permissions, func(i, j int) bool { return permissions[i] < permissions[j] })
		result[code] = permissions
	}
	return result, nil
}

// ReplaceRolePermissions replaces a role's permission set.
func (r *MemoryRepository) ReplaceRolePermissions(
	_ context.Context,
	code string,
	permissions []Permission,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	set := make(map[Permission]struct{}, len(permissions))
	for _, permission := range permissions {
		set[permission] = struct{}{}
	}
	r.rolePerms[code] = set
	return nil
}

func (r *MemoryRepository) CountRoleAssignments(_ context.Context, code string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, role := range r.userRoles {
		if role == code {
			count++
		}
	}
	return count, nil
}

func (r *MemoryRepository) UserOverrides(_ context.Context, publicID string) ([]Override, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	set := r.overrides[publicID]
	overrides := make([]Override, 0, len(set))
	for _, override := range set {
		overrides = append(overrides, override)
	}
	sort.Slice(
		overrides,
		func(i, j int) bool { return overrides[i].Permission < overrides[j].Permission },
	)
	return overrides, nil
}

// UserOverridesMany returns overrides grouped by user in one pass.
func (r *MemoryRepository) UserOverridesMany(
	_ context.Context,
	publicIDs []string,
) (map[string][]Override, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make(map[string][]Override, len(publicIDs))
	for _, publicID := range publicIDs {
		set := r.overrides[publicID]
		overrides := make([]Override, 0, len(set))
		for _, override := range set {
			overrides = append(overrides, override)
		}
		sort.Slice(
			overrides,
			func(i, j int) bool { return overrides[i].Permission < overrides[j].Permission },
		)
		result[publicID] = overrides
	}
	return result, nil
}

// SetUserOverride upserts a user's override.
func (r *MemoryRepository) SetUserOverride(
	_ context.Context,
	input SetOverrideInput,
) (Override, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.userRoles[input.UserPublicID]; !ok {
		return Override{}, ErrUserNotFound
	}
	override := Override{
		Permission: input.Permission,
		Effect:     input.Effect,
		Reason:     input.Reason,
		ExpiresAt:  input.ExpiresAt,
	}
	if r.overrides[input.UserPublicID] == nil {
		r.overrides[input.UserPublicID] = make(map[Permission]Override)
	}
	r.overrides[input.UserPublicID][input.Permission] = override
	return override, nil
}

func (r *MemoryRepository) DeleteUserOverride(
	_ context.Context,
	publicID string,
	permission Permission,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.overrides[publicID], permission)
	return nil
}

func (r *MemoryRepository) UserRole(_ context.Context, publicID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.userRoles[publicID], nil
}

func (r *MemoryRepository) AssignUserRole(_ context.Context, publicID, role string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userRoles[publicID] = role
	return nil
}
