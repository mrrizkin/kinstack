package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"kinstack/internal/platform/i18n"
	platformoauth "kinstack/internal/platform/oauth"
)

var (
	// ErrOAuthUnavailable identifies a disabled or unreachable SSO provider.
	ErrOAuthUnavailable = errors.New("oauth authentication is unavailable")
	// ErrOAuthUnregistered identifies a provider identity without a local account.
	ErrOAuthUnregistered = errors.New(i18n.T("auth.oauth_unregistered"))
	// ErrOAuthEmailUnverified identifies an identity whose email is not trusted.
	ErrOAuthEmailUnverified = errors.New(i18n.T("auth.oauth_email_unverified"))
)

// SAMLSignIn resolves an identity asserted by a SAML IdP to an existing
// account session, sharing matching and error semantics with OAuth sign-in.
func (s *Service) SAMLSignIn(
	ctx context.Context,
	identity platformoauth.Identity,
) (SessionResponse, error) {
	return s.OAuthSignIn(ctx, identity)
}

// bindIdentity links a provider subject to an existing account on first
// sign-in. Only a verified email that already belongs to an active account is
// accepted; the base never provisions accounts from an external provider.
func (s *Service) bindIdentity(
	ctx context.Context,
	identity platformoauth.Identity,
	provider string,
	subject string,
) (FindUserByUsernameResult, error) {
	email := strings.ToLower(strings.TrimSpace(identity.Email))
	parsed, err := mail.ParseAddress(email)
	if !identity.EmailVerified || email == "" || err != nil ||
		parsed.Address != email || len(email) > 320 {
		return FindUserByUsernameResult{}, ErrOAuthEmailUnverified
	}
	result, err := s.repository.FindUserByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return FindUserByUsernameResult{}, ErrOAuthUnregistered
	}
	if err != nil {
		return FindUserByUsernameResult{}, err
	}
	if err := s.repository.UpsertIdentity(ctx, result.PublicID, provider, subject, email); err != nil {
		if errors.Is(err, ErrIdentityConflict) || errors.Is(err, ErrNotFound) {
			return FindUserByUsernameResult{}, ErrOAuthUnregistered
		}
		return FindUserByUsernameResult{}, err
	}
	return result, nil
}

// OAuthSignIn resolves a provider identity to an existing account session. The
// base never auto-provisions accounts from an external provider: only a
// verified email that already belongs to an active account is accepted.
func (s *Service) OAuthSignIn(
	ctx context.Context,
	identity platformoauth.Identity,
) (SessionResponse, error) {
	provider := strings.ToLower(strings.TrimSpace(identity.Provider))
	subject := strings.TrimSpace(identity.Subject)
	if provider == "" || subject == "" {
		return SessionResponse{}, ErrOAuthUnregistered
	}

	// A bound (provider, subject) pair is authoritative: the provider's email
	// may change without invalidating the account binding.
	result, err := s.repository.FindIdentityUser(ctx, provider, subject)
	if errors.Is(err, ErrNotFound) {
		result, err = s.bindIdentity(ctx, identity, provider, subject)
	}
	if err != nil {
		return SessionResponse{}, err
	}
	if !result.IsActive {
		return SessionResponse{}, ErrInactiveUser
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
