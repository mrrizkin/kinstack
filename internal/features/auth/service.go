package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	platformactor "kinstack/internal/platform/actor"
	"kinstack/internal/platform/clock"
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
	Now        func() time.Time
}

// CredentialRevoker revokes every session and API token after a password reset.
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
	now        func() time.Time
}

func NewService(dependency ServiceDependency) *Service {
	assert.Assert(dependency.Repository != nil, "auth repository is required")

	now := dependency.Now
	if now == nil {
		now = clock.System
	}

	return &Service{
		repository: dependency.Repository,
		accessKey:  strings.TrimSpace(dependency.AccessKey),
		secretKey:  dependency.SecretKey,
		email:      dependency.Email,
		emailFrom:  strings.TrimSpace(dependency.EmailFrom),
		baseURL:    strings.TrimRight(strings.TrimSpace(dependency.BaseURL), "/"),
		lockout:    dependency.Lockout,
		revoker:    dependency.Revoker,
		now:        now,
	}
}

func (s *Service) SecondFactorAllowed(ctx context.Context, userPublicID, client string) bool {
	if s.lockout == nil {
		return true
	}
	return s.lockout.AllowedPair(ctx, secondFactorAccount(userPublicID), client)
}

func (s *Service) RecordSecondFactorFailure(ctx context.Context, userPublicID, client string) {
	if s.lockout == nil {
		return
	}
	s.lockout.Failure(ctx, secondFactorAccount(userPublicID), client, true)
}

func (s *Service) ClearSecondFactorFailures(ctx context.Context, userPublicID, client string) {
	if s.lockout == nil {
		return
	}
	s.lockout.Success(ctx, secondFactorAccount(userPublicID), client)
}

func secondFactorAccount(userPublicID string) string {
	return "2fa:" + strings.TrimSpace(userPublicID)
}

// Login verifies credentials and returns the session actor; every failure returns the
// same error. The environment ACCESS_KEY/SECRET_KEY pair is a break-glass super admin,
// checked before the database and bypassing RBAC.
func (s *Service) Login(
	ctx context.Context,
	username string,
	rawPassword string,
) (SessionResponse, error) {
	return s.LoginFrom(ctx, username, rawPassword, "")
}

// LoginFrom throttles per (account, client), so one client cannot lock a known account.
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
	if !s.lockoutAllows(ctx, username, client, breakGlass) {
		return SessionResponse{}, ErrAccountLocked
	}
	if s.isSuperAdmin(username, rawPassword) {
		s.recordLoginSuccess(ctx, username, client)
		return s.superAdminSession(username), nil
	}
	result, err := s.repository.FindUserByUsername(ctx, username)
	if errors.Is(err, ErrNotFound) {
		// Hash a dummy value so timing does not reveal whether the username exists.
		_ = platformpassword.Compare(platformpassword.DummyHash(), rawPassword)
		return SessionResponse{}, s.rejectLogin(ctx, username, client, ErrInvalidCredentials)
	}
	if err != nil {
		return SessionResponse{}, err
	}
	if err := platformpassword.Compare(result.PasswordHash, rawPassword); err != nil {
		return SessionResponse{}, s.rejectLogin(ctx, username, client, ErrInvalidCredentials)
	}
	if !result.IsActive {
		return SessionResponse{}, s.rejectLogin(ctx, username, client, ErrInactiveUser)
	}
	s.recordLoginSuccess(ctx, username, client)
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

// lockoutAllows is the only place the lockout question is asked; break-glass skips the
// account-wide lock and keeps its own pair limit.
func (s *Service) lockoutAllows(
	ctx context.Context,
	username, client string,
	breakGlass bool,
) bool {
	if s.lockout == nil {
		return true
	}
	allowed := s.lockout.AllowedPair(ctx, username, client)
	if allowed && !breakGlass {
		allowed = !s.lockout.AccountLocked(ctx, username)
	}
	return allowed
}

// rejectLogin counts the failure; absent user, wrong password and inactive account count
// the same so neither timing nor lockout reveals which one it was.
func (s *Service) rejectLogin(
	ctx context.Context,
	username, client string,
	reason error,
) error {
	if s.lockout != nil {
		s.lockout.Failure(ctx, username, client, true)
	}
	return reason
}

func (s *Service) recordLoginSuccess(ctx context.Context, username, client string) {
	if s.lockout != nil {
		s.lockout.Success(ctx, username, client)
	}
}

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

func (s *Service) isSuperAdmin(username, rawPassword string) bool {
	if s.accessKey == "" || s.secretKey == "" {
		return false
	}
	usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(s.accessKey)) == 1
	passwordMatch := subtle.ConstantTimeCompare([]byte(rawPassword), []byte(s.secretKey)) == 1
	return usernameMatch && passwordMatch
}

func (s *Service) superAdminSession(username string) SessionResponse {
	return SessionResponse{
		PublicID:    platformactor.SuperuserPublicID,
		Username:    username,
		DisplayName: "Super Admin",
		Superuser:   true,
	}
}
