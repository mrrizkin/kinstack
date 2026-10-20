package users

import (
	"context"
	"strings"
	"time"

	"kinstack/internal/platform/authz"
)

// RoleOptions returns every role as selectable options.
func (s *Service) RoleOptions(ctx context.Context) ([]RoleDTO, error) {
	authorization, err := s.requireAuthorization()
	if err != nil {
		return nil, err
	}
	roles, err := authorization.ListRoles(ctx)
	if err != nil {
		return nil, err
	}
	options := make([]RoleDTO, 0, len(roles))
	for _, role := range roles {
		options = append(options, roleToDTO(role, nil))
	}
	return options, nil
}

// AccessView builds the access management payload: roles with their permission
// sets, the permission catalog, and users with their overrides.
func (s *Service) AccessView(ctx context.Context) (AccessPageProps, error) {
	authorization, err := s.requireAuthorization()
	if err != nil {
		return AccessPageProps{}, err
	}
	roles, err := authorization.ListRoles(ctx)
	if err != nil {
		return AccessPageProps{}, err
	}
	roleDTOs := make([]RoleDTO, 0, len(roles))
	roleNames := make(map[string]string, len(roles))
	for _, role := range roles {
		permissions, err := authorization.RolePermissions(ctx, role.Code)
		if err != nil {
			return AccessPageProps{}, err
		}
		roleNames[role.Code] = role.Name
		roleDTOs = append(roleDTOs, roleToDTO(role, permissions))
	}

	definitions := authz.SortByCategory()
	permissionDTOs := make([]PermissionDTO, 0, len(definitions))
	for _, definition := range definitions {
		permissionDTOs = append(permissionDTOs, PermissionDTO{
			Code:        string(definition.Code),
			Category:    definition.Category,
			Label:       definition.Label,
			Description: definition.Description,
		})
	}

	users, err := s.repository.ListAllUsers(ctx)
	if err != nil {
		return AccessPageProps{}, err
	}
	userDTOs := make([]UserAccessDTO, 0, len(users))
	now := time.Now()
	for _, user := range users {
		overrides, err := authorization.UserOverrides(ctx, user.PublicID)
		if err != nil {
			return AccessPageProps{}, err
		}
		userDTOs = append(userDTOs, UserAccessDTO{
			PublicID:    user.PublicID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        user.Role,
			RoleName:    roleNames[user.Role],
			IsActive:    user.IsActive,
			Overrides:   overridesToDTO(overrides, now),
		})
	}

	return AccessPageProps{
		Roles:       roleDTOs,
		Permissions: permissionDTOs,
		Categories:  authz.CategoryOrder(),
		Users:       userDTOs,
	}, nil
}

// CreateRole validates and persists a new role.
func (s *Service) CreateRole(ctx context.Context, request CreateRoleRequest) (RoleDTO, error) {
	authorization, err := s.requireAuthorization()
	if err != nil {
		return RoleDTO{}, err
	}
	role, err := authorization.CreateRole(ctx, request.Code, request.Name, request.Description, request.Rank)
	if err != nil {
		return RoleDTO{}, err
	}
	return roleToDTO(role, nil), nil
}

// UpdateRole edits a custom role.
func (s *Service) UpdateRole(ctx context.Context, code string, request UpdateRoleRequest) (RoleDTO, error) {
	authorization, err := s.requireAuthorization()
	if err != nil {
		return RoleDTO{}, err
	}
	role, err := authorization.UpdateRole(ctx, code, request.Name, request.Description, request.Rank)
	if err != nil {
		return RoleDTO{}, err
	}
	return roleToDTO(role, nil), nil
}

// DeleteRole removes a custom role that is not assigned to any user.
func (s *Service) DeleteRole(ctx context.Context, code string) error {
	authorization, err := s.requireAuthorization()
	if err != nil {
		return err
	}
	return authorization.DeleteRole(ctx, code)
}

// SetRolePermissions replaces a role's permission set.
func (s *Service) SetRolePermissions(ctx context.Context, code string, permissions []string) error {
	authorization, err := s.requireAuthorization()
	if err != nil {
		return err
	}
	codes := make([]authz.Permission, 0, len(permissions))
	for _, permission := range permissions {
		codes = append(codes, authz.Permission(strings.TrimSpace(permission)))
	}
	return authorization.ReplaceRolePermissions(ctx, code, codes)
}

// AssignRole changes a user's role.
func (s *Service) AssignRole(ctx context.Context, publicID, role string) error {
	authorization, err := s.requireAuthorization()
	if err != nil {
		return err
	}
	return authorization.AssignRole(ctx, publicID, role)
}

// SetOverride validates and upserts a per-user permission override.
func (s *Service) SetOverride(ctx context.Context, publicID string, request SetOverrideRequest) error {
	authorization, err := s.requireAuthorization()
	if err != nil {
		return err
	}
	expiresAt, err := parseExpiry(request.ExpiresAt)
	if err != nil {
		return err
	}
	_, err = authorization.SetOverride(
		ctx,
		publicID,
		authz.Permission(strings.TrimSpace(request.Permission)),
		authz.Effect(strings.TrimSpace(request.Effect)),
		request.Reason,
		expiresAt,
	)
	return err
}

// DeleteOverride removes a per-user permission override.
func (s *Service) DeleteOverride(ctx context.Context, publicID, permission string) error {
	authorization, err := s.requireAuthorization()
	if err != nil {
		return err
	}
	return authorization.DeleteOverride(ctx, publicID, authz.Permission(strings.TrimSpace(permission)))
}

func (s *Service) requireAuthorization() (*authz.Service, error) {
	if s.authorization == nil {
		return nil, ErrAuthorizationUnavailable
	}
	return s.authorization, nil
}

func roleToDTO(role authz.Role, permissions []authz.Permission) RoleDTO {
	codes := make([]string, 0, len(permissions))
	for _, permission := range permissions {
		codes = append(codes, string(permission))
	}
	return RoleDTO{
		Code:        role.Code,
		Name:        role.Name,
		Description: role.Description,
		Rank:        role.Rank,
		IsSystem:    role.IsSystem,
		Permissions: codes,
	}
}

func overridesToDTO(overrides []authz.Override, now time.Time) []OverrideDTO {
	items := make([]OverrideDTO, 0, len(overrides))
	for _, override := range overrides {
		label := string(override.Permission)
		if definition, ok := authz.Lookup(override.Permission); ok {
			label = definition.Label
		}
		item := OverrideDTO{
			Permission: string(override.Permission),
			Label:      label,
			Effect:     string(override.Effect),
			Reason:     override.Reason,
			Expired:    override.Expired(now),
		}
		if override.ExpiresAt != nil {
			item.ExpiresAt = override.ExpiresAt.UTC().Format(time.RFC3339)
		}
		items = append(items, item)
	}
	return items
}

func parseExpiry(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		end := parsed.Add(24*time.Hour - time.Second)
		return &end, nil
	}
	return nil, ErrInvalidInput
}
