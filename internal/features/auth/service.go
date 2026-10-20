package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"

	"kinstack/internal/pkg/assert"
	platformactor "kinstack/internal/platform/actor"
	"kinstack/internal/platform/email"
	platformpassword "kinstack/internal/platform/password"
)

// ServiceDependency contains the persistence boundary, environment super admin
// credentials, and outbound-email boundary used by the service.
type ServiceDependency struct {
	Repository Repository
	AccessKey  string
	SecretKey  string
	Email      email.Sender
	EmailFrom  string
	BaseURL    string
	Lockout    *Lockout
}

// Service owns the credential verification contract.
type Service struct {
	repository Repository
	accessKey  string
	secretKey  string
	email      email.Sender
	emailFrom  string
	baseURL    string
	lockout    *Lockout
}

// NewService constructs an auth service around its persistence contract.
func NewService(dependency ServiceDependency) *Service {
	assert.Assert(dependency.Repository != nil, "auth repository is required")

	return &Service{
		repository: dependency.Repository,
		accessKey:  strings.TrimSpace(dependency.AccessKey),
		secretKey:  dependency.SecretKey,
		email:      dependency.Email,
		emailFrom:  strings.TrimSpace(dependency.EmailFrom),
		baseURL:    strings.TrimRight(strings.TrimSpace(dependency.BaseURL), "/"),
		lockout:    dependency.Lockout,
	}
}

// Login verifies credentials and returns the authenticated session actor. Every
// credential failure returns the same error so callers cannot enumerate users.
//
// The environment ACCESS_KEY/SECRET_KEY pair is a break-glass super admin. Like
// HTTP basic auth it is checked before the database and has no stored account:
// the returned session is ephemeral and bypasses RBAC entirely.
func (s *Service) Login(
	ctx context.Context,
	username string,
	rawPassword string,
) (SessionResponse, error) {
	username = strings.TrimSpace(username)
	if username == "" || rawPassword == "" {
		return SessionResponse{}, ErrInvalidInput
	}
	if s.lockout != nil && !s.lockout.Allowed(ctx, username) {
		return SessionResponse{}, ErrAccountLocked
	}

	if s.isSuperAdmin(username, rawPassword) {
		if s.lockout != nil {
			s.lockout.Success(ctx, username)
		}
		return s.superAdminSession(username), nil
	}

	result, err := s.repository.FindUserByUsername(ctx, username)
	if errors.Is(err, ErrNotFound) {
		// Hash a dummy value so response timing does not reveal whether the
		// username exists.
		_ = platformpassword.Compare(platformpassword.DummyHash(), rawPassword)
		if s.lockout != nil {
			s.lockout.Failure(ctx, username)
		}
		return SessionResponse{}, ErrInvalidCredentials
	}
	if err != nil {
		return SessionResponse{}, err
	}
	if err := platformpassword.Compare(result.PasswordHash, rawPassword); err != nil {
		if s.lockout != nil {
			s.lockout.Failure(ctx, username)
		}
		return SessionResponse{}, ErrInvalidCredentials
	}
	if !result.IsActive {
		return SessionResponse{}, ErrInactiveUser
	}
	if s.lockout != nil {
		s.lockout.Success(ctx, username)
	}
	if err := s.repository.TouchLastLogin(ctx, result.PublicID); err != nil {
		return SessionResponse{}, err
	}
	return SessionResponse{
		PublicID:    result.PublicID,
		Username:    result.Username,
		DisplayName: result.DisplayName,
		Role:        result.Role,
	}, nil
}

// Session loads the authenticated session actor for an account that has
// already passed credential and second-factor checks.
func (s *Service) Session(ctx context.Context, publicID string) (SessionResponse, error) {
	result, err := s.repository.FindUserByPublicID(ctx, publicID)
	if errors.Is(err, ErrNotFound) {
		return SessionResponse{}, ErrInvalidCredentials
	}
	if err != nil {
		return SessionResponse{}, err
	}
	if !result.IsActive {
		return SessionResponse{}, ErrInactiveUser
	}
	return SessionResponse{
		PublicID:    result.PublicID,
		Username:    result.Username,
		DisplayName: result.DisplayName,
		Role:        result.Role,
	}, nil
}

// isSuperAdmin reports whether the credentials match the environment pair.
func (s *Service) isSuperAdmin(username, rawPassword string) bool {
	if s.accessKey == "" || s.secretKey == "" {
		return false
	}
	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(s.accessKey)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(rawPassword), []byte(s.secretKey)) == 1
	return usernameMatch && passwordMatch
}

// superAdminSession builds the ephemeral break-glass session. Nothing is
// persisted, so the operator can always sign in regardless of database state.
func (s *Service) superAdminSession(username string) SessionResponse {
	return SessionResponse{
		PublicID:    platformactor.SuperuserPublicID,
		Username:    username,
		DisplayName: "Super Admin",
		Superuser:   true,
	}
}
