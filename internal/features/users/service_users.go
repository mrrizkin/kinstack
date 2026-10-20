package users

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"kinstack/internal/pkg/pagination"
	"kinstack/internal/pkg/publicid"
	"kinstack/internal/platform/authz"
	platformpassword "kinstack/internal/platform/password"
)

// Page size bounds for the users list.
const (
	DefaultPerPage = 25
	MaxPerPage     = 100
)

// ParsePageQuery validates raw page/per-page query strings. Invalid values are
// rejected rather than clamped so the API stays fail-closed.
func ParsePageQuery(rawPage, rawPerPage string) (pagination.Query, error) {
	page, err := parsePositive(rawPage)
	if err != nil {
		return pagination.Query{}, err
	}
	perPage, err := parsePositive(rawPerPage)
	if err != nil {
		return pagination.Query{}, err
	}
	if perPage == 0 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		return pagination.Query{}, ErrInvalidInput
	}
	if page == 0 {
		page = 1
	}
	return pagination.Query{Page: page, PerPage: perPage}, nil
}

func parsePositive(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, ErrInvalidInput
	}
	return value, nil
}

// UserList is one page of accounts plus its display window.
type UserList struct {
	Users []UserDTO
	Page  pagination.Page
}

// List returns one page of active (non-deleted) accounts.
func (s *Service) List(ctx context.Context, query pagination.Query) (UserList, error) {
	results, err := s.repository.ListUsers(ctx, query)
	if err != nil {
		return UserList{}, err
	}
	counts, err := s.repository.CountUsers(ctx)
	if err != nil {
		return UserList{}, err
	}
	items := make([]UserDTO, 0, len(results))
	for _, result := range results {
		items = append(items, userToDTO(result))
	}
	return UserList{Users: items, Page: pagination.NewPage(query, counts.Total)}, nil
}

// ListAll returns every account for access management views.
func (s *Service) ListAll(ctx context.Context) ([]UserDTO, error) {
	results, err := s.repository.ListAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]UserDTO, 0, len(results))
	for _, result := range results {
		items = append(items, userToDTO(result))
	}
	return items, nil
}

// Count returns total and active account counts.
func (s *Service) Count(ctx context.Context) (UserCounts, error) {
	return s.repository.CountUsers(ctx)
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
	if s.revoker != nil {
		if err := s.revoker.RevokeAll(ctx, publicID); err != nil {
			return err
		}
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
