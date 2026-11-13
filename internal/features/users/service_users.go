package users

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"kinstack/internal/pkg/pagination"
	"kinstack/internal/pkg/publicid"
	"kinstack/internal/platform/authz"
	platformpassword "kinstack/internal/platform/password"
)

// Page size bounds for the users list.
func ParsePageQuery(rawPage, rawPerPage string, fallbackPerPage int) (pagination.Query, error) {
	query, err := pagination.ParseRequest(rawPage, rawPerPage, fallbackPerPage)
	if err != nil {
		return pagination.Query{}, ErrInvalidInput
	}
	return query, nil
}

type UserList struct {
	Users []UserDTO
	Page  pagination.Page
}

func (s *Service) List(ctx context.Context, query pagination.Query) (UserList, error) {
	counts, err := s.repository.CountUsers(ctx)
	if err != nil {
		return UserList{}, err
	}
	query = pagination.ClampPage(query, counts.Total)
	results, err := s.repository.ListUsers(ctx, query)
	if err != nil {
		return UserList{}, err
	}
	items := make([]UserDTO, 0, len(results))
	for _, result := range results {
		items = append(items, userToDTO(result))
	}
	return UserList{Users: items, Page: pagination.NewPage(query, counts.Total)}, nil
}

func (s *Service) Count(ctx context.Context) (UserCounts, error) {
	return s.repository.CountUsers(ctx)
}

func (s *Service) Create(ctx context.Context, request CreateUserRequest) (UserDTO, error) {
	username := strings.TrimSpace(request.Username)
	displayName := strings.TrimSpace(request.DisplayName)
	role := strings.TrimSpace(request.Role)
	email := normalizeEmail(request.Email)

	if !usernamePattern.MatchString(username) || len(username) > usernameLengthMax {
		return UserDTO{}, ErrInvalidInput
	}
	if email != "" && (!emailPattern.MatchString(email) || len(email) > emailLengthMax) {
		return UserDTO{}, ErrInvalidInput
	}
	if displayName == "" {
		return UserDTO{}, ErrInvalidInput
	}
	if len(request.Password) < passwordLengthMin {
		return UserDTO{}, ErrInvalidInput
	}
	if err := s.ensureRole(ctx, role); err != nil {
		return UserDTO{}, err
	}
	if err := s.authorizeRoleGrant(ctx, role); err != nil {
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

// Invite stores an unusable password hash, so the emailed link is the only way in.
func (s *Service) Invite(ctx context.Context, request InviteUserRequest) (UserDTO, error) {
	invite, err := normalizeInvite(request)
	if err != nil {
		return UserDTO{}, err
	}
	if err := s.ensureRole(ctx, invite.role); err != nil {
		return UserDTO{}, err
	}
	if err := s.authorizeRoleGrant(ctx, invite.role); err != nil {
		return UserDTO{}, err
	}
	passwordHash, err := invitationPasswordHash()
	if err != nil {
		return UserDTO{}, err
	}
	result, err := s.createInvitedUser(ctx, invite, passwordHash)
	if err != nil {
		return UserDTO{}, err
	}
	if err := s.authorization.AssignRole(ctx, result.PublicID, invite.role); err != nil {
		return UserDTO{}, err
	}
	result.Role = invite.role
	if s.inviter != nil {
		if err := s.inviter.SendInvitation(ctx, invite.email); err != nil {
			return userToDTO(result), fmt.Errorf("send invitation: %w", err)
		}
	}
	return userToDTO(result), nil
}

type inviteRequest struct {
	email        string
	displayName  string
	role         string
	usernameBase string
}

func normalizeInvite(request InviteUserRequest) (inviteRequest, error) {
	email := normalizeEmail(request.Email)
	if email == "" || !emailPattern.MatchString(email) || len(email) > emailLengthMax {
		return inviteRequest{}, ErrInvalidInput
	}
	displayName := strings.TrimSpace(request.DisplayName)
	if displayName == "" {
		displayName = email
	}
	usernameBase := strings.TrimSpace(request.Username)
	if usernameBase == "" {
		usernameBase = usernameFromEmail(email)
	}
	return inviteRequest{
		email:        email,
		displayName:  displayName,
		role:         strings.TrimSpace(request.Role),
		usernameBase: usernameBase,
	}, nil
}

func (s *Service) createInvitedUser(
	ctx context.Context,
	invite inviteRequest,
	passwordHash string,
) (UserResult, error) {
	var result UserResult
	for attempt := 0; attempt < inviteUsernameAttempts; attempt++ {
		username := invite.usernameBase
		if attempt > 0 {
			suffix := publicid.MustNew()
			username = trimToLength(invite.usernameBase+"-"+suffix[:4], usernameLengthMax)
		}
		created, err := s.repository.CreateUser(ctx, CreateUserInput{
			PublicID:     publicid.MustNew(),
			Username:     username,
			Email:        invite.email,
			DisplayName:  invite.displayName,
			PasswordHash: passwordHash,
			Invited:      true,
		})
		if errors.Is(err, ErrEmailTaken) {
			return UserResult{}, ErrEmailTaken
		}
		if errors.Is(err, ErrUsernameTaken) {
			continue
		}
		if err != nil {
			return UserResult{}, err
		}
		result = created
		break
	}
	if result.PublicID == "" {
		return UserResult{}, ErrUsernameTaken
	}
	return result, nil
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
	return trimToLength(filtered, usernameLengthMax)
}

func trimToLength(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

// invitationPasswordHash hashes a discarded random secret: password sign-in stays
// impossible until the invitee sets one.
func invitationPasswordHash() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return platformpassword.Hash(base64.RawURLEncoding.EncodeToString(secret))
}

func (s *Service) Update(ctx context.Context,
	publicID string,
	request UpdateUserRequest,
) (UserDTO, error) {
	update, err := normalizeUserUpdate(request)
	if err != nil {
		return UserDTO{}, err
	}
	if err := s.ensureRole(ctx, update.role); err != nil {
		return UserDTO{}, err
	}
	existing, err := s.repository.FindUserByPublicID(ctx, publicID)
	if err != nil {
		return UserDTO{}, mapUserError(err)
	}
	if existing.Role != update.role {
		if err := s.authorizeRoleGrant(ctx, update.role); err != nil {
			return UserDTO{}, err
		}
		if err := s.authorizeUserChange(ctx, publicID); err != nil {
			return UserDTO{}, err
		}
	}
	result, err := s.repository.UpdateUser(ctx, UpdateUserInput{
		PublicID:    publicID,
		Email:       update.email,
		DisplayName: update.displayName,
	})
	if err != nil {
		return UserDTO{}, mapUserError(err)
	}
	if err := s.authorization.AssignRole(ctx, publicID, update.role); err != nil {
		return UserDTO{}, err
	}
	result.Role = update.role
	return userToDTO(result), nil
}

type userUpdate struct {
	displayName string
	role        string
	email       string
}

func normalizeUserUpdate(request UpdateUserRequest) (userUpdate, error) {
	update := userUpdate{
		displayName: strings.TrimSpace(request.DisplayName),
		role:        strings.TrimSpace(request.Role),
		email:       normalizeEmail(request.Email),
	}
	if update.displayName == "" {
		return userUpdate{}, ErrInvalidInput
	}
	if update.email != "" &&
		(!emailPattern.MatchString(update.email) || len(update.email) > emailLengthMax) {
		return userUpdate{}, ErrInvalidInput
	}
	return update, nil
}

// mapUserError narrows repository errors to the ones this slice exposes. An
// unexpected error keeps its identity so it is logged once, upstream.
func mapUserError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return ErrNotFound
	case errors.Is(err, ErrEmailTaken):
		return ErrEmailTaken
	default:
		return err
	}
}

func (s *Service) ResetPassword(ctx context.Context, publicID, rawPassword string) error {
	if len(rawPassword) < passwordLengthMin {
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
			result, err := s.repository.DeactivateGuarded(ctx, publicID)
			switch {
			case errors.Is(err, ErrLastAdministrator):
				return UserDTO{}, ErrLastAdministrator
			case errors.Is(err, ErrNotFound):
				return UserDTO{}, ErrNotFound
			case err != nil:
				return UserDTO{}, err
			}
			return userToDTO(result), nil
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

// authorizeRoleGrant reads the acting principal from the request context, which handlers
// attach once per request.
func (s *Service) authorizeRoleGrant(ctx context.Context, roleCode string) error {
	if s.authorization == nil {
		return nil
	}
	if err := s.authorization.AuthorizeRoleGrant(ctx, roleCode); err != nil {
		return mapAuthorizationError(err)
	}
	return nil
}

func (s *Service) authorizeUserChange(ctx context.Context, publicID string) error {
	if s.authorization == nil {
		return nil
	}
	if err := s.authorization.AuthorizeUserChange(ctx, publicID); err != nil {
		return mapAuthorizationError(err)
	}
	return nil
}

func mapAuthorizationError(err error) error {
	switch {
	case errors.Is(err, authz.ErrSelfChange):
		return ErrSelfChange
	case errors.Is(err, authz.ErrPrivilegeEscalation),
		errors.Is(err, authz.ErrManagementSession):
		return ErrForbidden
	case errors.Is(err, authz.ErrSystemRole):
		return ErrSystemRole
	default:
		return err
	}
}
