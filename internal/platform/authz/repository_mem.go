package authz

import (
	"context"
	"sort"
	"sync"
	"time"
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

// NewMemoryRepository constructs an empty in-memory repository.
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

// ListRoles returns every role.
func (r *MemoryRepository) ListRoles(context.Context) ([]Role, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	roles := make([]Role, 0, len(r.roles))
	for _, role := range r.roles {
		roles = append(roles, role)
	}
	sort.Slice(roles, func(i, j int) bool { return roles[i].Code < roles[j].Code })
	return roles[:min(len(roles), 100)], nil
}

// CountRoles returns the number of roles.
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

// CreateRole inserts a new role.
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

// DeleteRole removes a role.
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

// RolePermissions returns a role's permission set.
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

// CountRoleAssignments counts users holding a role.
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

// UserOverrides returns a user's overrides.
func (r *MemoryRepository) UserOverrides(_ context.Context, publicID string) ([]Override, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	set := r.overrides[publicID]
	overrides := make([]Override, 0, len(set))
	for _, override := range set {
		overrides = append(overrides, override)
	}
	sort.Slice(overrides, func(i, j int) bool { return overrides[i].Permission < overrides[j].Permission })
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
		sort.Slice(overrides, func(i, j int) bool { return overrides[i].Permission < overrides[j].Permission })
		result[publicID] = overrides
	}
	return result, nil
}

// SetUserOverride upserts a user's override.
func (r *MemoryRepository) SetUserOverride(_ context.Context, input SetOverrideInput) (Override, error) {
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

// DeleteUserOverride removes a user's override.
func (r *MemoryRepository) DeleteUserOverride(_ context.Context, publicID string, permission Permission) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.overrides[publicID], permission)
	return nil
}

// UserRole returns the role assigned to one user, or "" when unassigned.
func (r *MemoryRepository) UserRole(_ context.Context, publicID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.userRoles[publicID], nil
}

// AssignUserRole changes a user's role.
func (r *MemoryRepository) AssignUserRole(_ context.Context, publicID, role string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.userRoles[publicID] = role
	return nil
}
