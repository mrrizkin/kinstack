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

type ServiceDependency struct {
	Repository Repository
	AccessKey  string
	SecretKey  string
	Email      email.Sender
	EmailFrom  string
	BaseURL    string
	Lockout    *Lockout
	Revoker    CredentialRevoker
}

// CredentialRevoker revokes every browser session and personal API token of a
// user. It is invoked after a password reset so old credentials cannot outlive
// the new password.
type CredentialRevoker interface {
	RevokeAll(ctx context.Context, userPublicID string) error
}

type Service struct {
	repository Repository
	accessKey  string
	secretKey  string
	email      email.Sender
	emailFrom  string
	baseURL    string
	lockout    *Lockout
	revoker    CredentialRevoker
}

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
		revoker:    dependency.Revoker,
	}
}

// SecondFactorAllowed reports whether the account may still attempt a second
// factor from this client. It uses a separate key namespace from password
// attempts so a 2FA lockout never blocks a fresh sign-in.
func (s *Service) SecondFactorAllowed(ctx context.Context, userPublicID, client string) bool {
	if s.lockout == nil {
		return true
	}
	return s.lockout.AllowedPair(ctx, secondFactorAccount(userPublicID), client)
}

// RecordSecondFactorFailure counts one rejected second-factor attempt. A 2FA
// code is short, so the account-wide counter also applies to slow distributed
// guessing.
func (s *Service) RecordSecondFactorFailure(ctx context.Context, userPublicID, client string) {
	if s.lockout == nil {
		return
	}
	s.lockout.Failure(ctx, secondFactorAccount(userPublicID), client, true)
}

// ClearSecondFactorFailures resets the counter after a successful challenge.
func (s *Service) ClearSecondFactorFailures(ctx context.Context, userPublicID, client string) {
	if s.lockout == nil {
		return
	}
	s.lockout.Success(ctx, secondFactorAccount(userPublicID), client)
}

// secondFactorAccount namespaces lockout keys for the second-factor challenge.
func secondFactorAccount(userPublicID string) string {
	return "2fa:" + strings.TrimSpace(userPublicID)
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
	return s.LoginFrom(ctx, username, rawPassword, "")
}

// LoginFrom is Login with the calling client (usually the remote IP). Failures
// are throttled per (account, client) pair, so one client cannot lock a known
// account for everyone else.
func (s *Service) LoginFrom(
	ctx context.Context,
	username string,
	rawPassword string,
	client string,
) (SessionResponse, error) {
	username = strings.TrimSpace(username)
	client = strings.TrimSpace(client)
	if username == "" || rawPassword == "" {
		return SessionResponse{}, ErrInvalidInput
	}
	breakGlass := s.accessKey != "" && strings.EqualFold(username, s.accessKey)
	if s.lockout != nil {
		allowed := s.lockout.AllowedPair(ctx, username, client)
		if allowed && !breakGlass {
			allowed = !s.lockout.AccountLocked(ctx, username)
		}
		if !allowed {
			return SessionResponse{}, ErrAccountLocked
		}
	}

	if s.isSuperAdmin(username, rawPassword) {
		if s.lockout != nil {
			s.lockout.Success(ctx, username, client)
		}
		return s.superAdminSession(username), nil
	}

	result, err := s.repository.FindUserByUsername(ctx, username)
	if errors.Is(err, ErrNotFound) {
		// Hash a dummy value so response timing does not reveal whether the
		// username exists.
		_ = platformpassword.Compare(platformpassword.DummyHash(), rawPassword)
		if s.lockout != nil {
			s.lockout.Failure(ctx, username, client, true)
		}
		return SessionResponse{}, ErrInvalidCredentials
	}
	if err != nil {
		return SessionResponse{}, err
	}
	if err := platformpassword.Compare(result.PasswordHash, rawPassword); err != nil {
		if s.lockout != nil {
			s.lockout.Failure(ctx, username, client, true)
		}
		return SessionResponse{}, ErrInvalidCredentials
	}
	if !result.IsActive {
		if s.lockout != nil {
			s.lockout.Failure(ctx, username, client, true)
		}
		return SessionResponse{}, ErrInactiveUser
	}
	if s.lockout != nil {
		s.lockout.Success(ctx, username, client)
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
