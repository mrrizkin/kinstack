package users

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/publicid"
	"kinstack/internal/platform/authz"
	platformpassword "kinstack/internal/platform/password"
)

const (
	minPasswordLength = 8
	maxUsernameLength = 32
	maxEmailLength    = 254
)

var (
	usernamePattern = regexp.MustCompile(`^[a-zA-Z0-9._-]{3,32}$`)
	emailPattern    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// Inviter delivers an invitation email to a newly created account.
type Inviter interface {
	SendInvitation(ctx context.Context, address string) error
}

// ServiceDependency contains the persistence boundary used by the service.
type ServiceDependency struct {
	Repository    Repository
	Authorization *authz.Service
	Inviter       Inviter
}

// Service owns user account business rules.
type Service struct {
	repository    Repository
	authorization *authz.Service
	inviter       Inviter
}

// NewService constructs a users service around its persistence contract.
func NewService(dependency ServiceDependency) *Service {
	assert.Assert(dependency.Repository != nil, "users repository is required")

	return &Service{
		repository:    dependency.Repository,
		authorization: dependency.Authorization,
		inviter:       dependency.Inviter,
	}
}

func (s *Service) ensureRole(ctx context.Context, role string) error {
	role = strings.TrimSpace(role)
	if role == "" {
		return ErrInvalidRole
	}
	if s.authorization == nil {
		return ErrAuthorizationUnavailable
	}
	if _, err := s.authorization.FindRole(ctx, role); err != nil {
		if errors.Is(err, authz.ErrRoleNotFound) {
			return ErrInvalidRole
		}
		return err
	}
	return nil
}

// List returns every active (non-deleted) account.
func (s *Service) List(ctx context.Context) ([]UserDTO, error) {
	results, err := s.repository.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]UserDTO, 0, len(results))
	for _, result := range results {
		items = append(items, userToDTO(result))
	}
	return items, nil
}

// Create validates and persists a new account.
func (s *Service) Create(ctx context.Context, request CreateUserRequest) (UserDTO, error) {
	username := strings.TrimSpace(request.Username)
	displayName := strings.TrimSpace(request.DisplayName)
	role := strings.TrimSpace(request.Role)
	email := normalizeEmail(request.Email)

	if !usernamePattern.MatchString(username) || len(username) > maxUsernameLength {
		return UserDTO{}, ErrInvalidInput
	}
	if email != "" && (!emailPattern.MatchString(email) || len(email) > maxEmailLength) {
		return UserDTO{}, ErrInvalidInput
	}
	if displayName == "" {
		return UserDTO{}, ErrInvalidInput
	}
	if len(request.Password) < minPasswordLength {
		return UserDTO{}, ErrInvalidInput
	}
	if err := s.ensureRole(ctx, role); err != nil {
		return UserDTO{}, err
	}

	hash, err := platformpassword.Hash(request.Password)
	if err != nil {
		return UserDTO{}, err
	}
	result, err := s.repository.CreateUser(ctx, CreateUserInput{
		PublicID:     publicid.MustNew(),
		Username:     username,
		Email:        email,
		DisplayName:  displayName,
		PasswordHash: hash,
	})
	if errors.Is(err, ErrUsernameTaken) {
		return UserDTO{}, ErrUsernameTaken
	}
	if errors.Is(err, ErrEmailTaken) {
		return UserDTO{}, ErrEmailTaken
	}
	if err != nil {
		return UserDTO{}, err
	}
	if err := s.authorization.AssignRole(ctx, result.PublicID, role); err != nil {
		return UserDTO{}, err
	}
	result.Role = role
	return userToDTO(result), nil
}

// Invite creates an account from an email address and sends a sign-in
// invitation. The stored password hash is unusable until the invitee sets a
// password, so the only way in is the emailed link.
func (s *Service) Invite(ctx context.Context, request InviteUserRequest) (UserDTO, error) {
	email := normalizeEmail(request.Email)
	displayName := strings.TrimSpace(request.DisplayName)
	role := strings.TrimSpace(request.Role)
	if email == "" || !emailPattern.MatchString(email) || len(email) > maxEmailLength {
		return UserDTO{}, ErrInvalidInput
	}
	if displayName == "" {
		displayName = email
	}
	if err := s.ensureRole(ctx, role); err != nil {
		return UserDTO{}, err
	}
	passwordHash, err := invitationPasswordHash()
	if err != nil {
		return UserDTO{}, err
	}
	usernameBase := strings.TrimSpace(request.Username)
	if usernameBase == "" {
		usernameBase = usernameFromEmail(email)
	}

	var result UserResult
	for attempt := 0; attempt < 5; attempt++ {
		username := usernameBase
		if attempt > 0 {
			suffix := publicid.MustNew()
			username = trimToLength(usernameBase+"-"+suffix[:4], maxUsernameLength)
		}
		result, err = s.repository.CreateUser(ctx, CreateUserInput{
			PublicID:     publicid.MustNew(),
			Username:     username,
			Email:        email,
			DisplayName:  displayName,
			PasswordHash: passwordHash,
			Invited:      true,
		})
		if errors.Is(err, ErrEmailTaken) {
			return UserDTO{}, ErrEmailTaken
		}
		if errors.Is(err, ErrUsernameTaken) {
			continue
		}
		if err != nil {
			return UserDTO{}, err
		}
		break
	}
	if result.PublicID == "" {
		return UserDTO{}, ErrUsernameTaken
	}
	if err := s.authorization.AssignRole(ctx, result.PublicID, role); err != nil {
		return UserDTO{}, err
	}
	result.Role = role
	if s.inviter != nil {
		if err := s.inviter.SendInvitation(ctx, email); err != nil {
			return userToDTO(result), fmt.Errorf("send invitation: %w", err)
		}
	}
	return userToDTO(result), nil
}

func usernameFromEmail(email string) string {
	local, _, _ := strings.Cut(email, "@")
	filtered := strings.Map(func(character rune) rune {
		switch {
		case character >= 'a' && character <= 'z',
			character >= 'A' && character <= 'Z',
			character >= '0' && character <= '9',
			character == '.', character == '_', character == '-':
			return character
		}
		return -1
	}, local)
	filtered = strings.TrimLeft(filtered, "._-")
	if len(filtered) < 3 {
		filtered = "user" + filtered
	}
	return trimToLength(filtered, maxUsernameLength)
}

func trimToLength(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

// invitationPasswordHash stores a hash of a discarded random secret so password
// sign-in stays impossible until the invitee sets their own password.
func invitationPasswordHash() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return platformpassword.Hash(base64.RawURLEncoding.EncodeToString(secret))
}

// Update edits an account's display name and role.
func (s *Service) Update(ctx context.Context,
	publicID string,
	request UpdateUserRequest,
) (UserDTO, error) {
	displayName := strings.TrimSpace(request.DisplayName)
	role := strings.TrimSpace(request.Role)
	email := normalizeEmail(request.Email)
	if displayName == "" {
		return UserDTO{}, ErrInvalidInput
	}
	if email != "" && (!emailPattern.MatchString(email) || len(email) > maxEmailLength) {
		return UserDTO{}, ErrInvalidInput
	}
	if err := s.ensureRole(ctx, role); err != nil {
		return UserDTO{}, err
	}
	result, err := s.repository.UpdateUser(ctx, UpdateUserInput{
		PublicID:    publicID,
		Email:       email,
		DisplayName: displayName,
	})
	if errors.Is(err, ErrNotFound) {
		return UserDTO{}, ErrNotFound
	}
	if errors.Is(err, ErrEmailTaken) {
		return UserDTO{}, ErrEmailTaken
	}
	if err != nil {
		return UserDTO{}, err
	}
	if err := s.authorization.AssignRole(ctx, publicID, role); err != nil {
		return UserDTO{}, err
	}
	result.Role = role
	return userToDTO(result), nil
}

// ResetPassword replaces an account password after validating its length.
func (s *Service) ResetPassword(ctx context.Context, publicID, rawPassword string) error {
	if len(rawPassword) < minPasswordLength {
		return ErrInvalidInput
	}
	hash, err := platformpassword.Hash(rawPassword)
	if err != nil {
		return err
	}
	if err := s.repository.UpdatePassword(ctx, publicID, hash); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

// SetActive enables or disables an account. The last active account that can
// manage access cannot be disabled.
func (s *Service) SetActive(
	ctx context.Context,
	publicID string,
	active bool,
) (UserDTO, error) {
	if !active && s.authorization != nil {
		current, err := s.repository.FindUserByPublicID(ctx, publicID)
		if errors.Is(err, ErrNotFound) {
			return UserDTO{}, ErrNotFound
		}
		if err != nil {
			return UserDTO{}, err
		}
		if current.IsActive && s.authorization.Can(
			ctx,
			principalOf(current),
			authz.PermissionAccessManage,
			nil,
		).Allowed {
			others, err := s.countActiveAccessManagers(ctx, publicID)
			if err != nil {
				return UserDTO{}, err
			}
			if others == 0 {
				return UserDTO{}, ErrLastAdministrator
			}
		}
	}
	result, err := s.repository.SetActive(ctx, publicID, active)
	if errors.Is(err, ErrNotFound) {
		return UserDTO{}, ErrNotFound
	}
	if err != nil {
		return UserDTO{}, err
	}
	return userToDTO(result), nil
}

func principalOf(user UserResult) authz.Principal {
	return authz.Principal{
		PublicID:    user.PublicID,
		Role:        user.Role,
		DisplayName: user.DisplayName,
	}
}

func (s *Service) countActiveAccessManagers(
	ctx context.Context,
	excludePublicID string,
) (int, error) {
	users, err := s.repository.ListUsers(ctx)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, user := range users {
		if user.PublicID == excludePublicID || !user.IsActive {
			continue
		}
		if s.authorization.Can(ctx, principalOf(user), authz.PermissionAccessManage, nil).Allowed {
			count++
		}
	}
	return count, nil
}

func userToDTO(result UserResult) UserDTO {
	dto := UserDTO{
		PublicID:    result.PublicID,
		Username:    result.Username,
		Email:       result.Email,
		DisplayName: result.DisplayName,
		Role:        result.Role,
		IsActive:    result.IsActive,
		Invited:     result.Invited,
		CreatedAt:   result.CreatedAt.UTC().Format(time.RFC3339),
	}
	if result.LastLoginAt != nil {
		dto.LastLoginAt = result.LastLoginAt.UTC().Format(time.RFC3339)
	}
	return dto
}

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

	users, err := s.repository.ListUsers(ctx)
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
