package users

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/platform/authz"
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

// CredentialRevoker revokes the browser sessions and personal API tokens of a
// user after an administrative password reset.
type CredentialRevoker interface {
	RevokeAll(ctx context.Context, userPublicID string) error
}

// ServiceDependency contains the persistence boundary used by the service.
type ServiceDependency struct {
	Repository    Repository
	Authorization *authz.Service
	Inviter       Inviter
	Revoker       CredentialRevoker
}

// Service owns user account business rules.
type Service struct {
	repository    Repository
	authorization *authz.Service
	inviter       Inviter
	revoker       CredentialRevoker
}

// NewService constructs a users service around its persistence contract.
func NewService(dependency ServiceDependency) *Service {
	assert.Assert(dependency.Repository != nil, "users repository is required")

	return &Service{
		repository:    dependency.Repository,
		authorization: dependency.Authorization,
		inviter:       dependency.Inviter,
		revoker:       dependency.Revoker,
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

func principalOf(user UserResult) authz.Principal {
	return authz.Principal{
		PublicID:    user.PublicID,
		Role:        user.Role,
		DisplayName: user.DisplayName,
	}
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
