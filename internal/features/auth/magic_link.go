package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"kinstack/internal/platform/email"
	"kinstack/internal/platform/i18n"
)

// magicLinkTTL bounds how long a passwordless sign-in token stays valid.
const magicLinkTTL = 15 * time.Minute

// invitationTTL bounds how long an invitation link stays valid.
const invitationTTL = 7 * 24 * time.Hour

// RequestMagicLink issues a single-use sign-in link for the account owning the
// email address. It never reveals whether the address belongs to an account.
func (s *Service) RequestMagicLink(ctx context.Context, address string) error {
	address = strings.ToLower(strings.TrimSpace(address))
	if address == "" || !strings.Contains(address, "@") {
		return ErrInvalidInput
	}

	result, err := s.repository.FindUserByEmail(ctx, address)
	if errors.Is(err, ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !result.IsActive {
		return nil
	}

	raw, hash, err := newMagicToken()
	if err != nil {
		return err
	}
	expiresAt := s.now().Add(magicLinkTTL)
	if err := s.repository.CreateMagicLinkToken(ctx, MagicLinkToken{
		UserPublicID: result.PublicID,
		TokenHash:    hash,
		ExpiresAt:    expiresAt,
	}); err != nil {
		return err
	}
	if s.email == nil || s.emailFrom == "" {
		return nil
	}

	link := s.baseURL + "/login/magic-link/" + raw
	html, err := s.email.Render("magic-link.html", map[string]string{
		"Link":      link,
		"ExpiresAt": expiresAt.Format("02 Jan 2006 15:04 MST"),
	})
	if err != nil {
		return err
	}
	return s.email.Send(ctx, email.Message{
		To:             result.Email,
		From:           s.emailFrom,
		Subject:        i18n.T("auth.magic_link_subject"),
		TextBody:       i18n.T("auth.magic_link_body", link),
		HTMLBody:       html,
		IdempotencyKey: "magic-link:" + result.PublicID + ":" + hash,
	})
}

// SendInvitation issues a single-use link that lets a freshly invited account
// set its password and sign in for the first time. Unlike RequestMagicLink it
// reports failures because the caller just created the account.
func (s *Service) SendInvitation(ctx context.Context, address string) error {
	address = strings.ToLower(strings.TrimSpace(address))
	if address == "" || !strings.Contains(address, "@") {
		return ErrInvalidInput
	}

	result, err := s.repository.FindUserByEmail(ctx, address)
	if err != nil {
		return err
	}
	if !result.IsActive {
		return ErrInactiveUser
	}

	raw, hash, err := newMagicToken()
	if err != nil {
		return err
	}
	expiresAt := s.now().Add(invitationTTL)
	if err := s.repository.CreateMagicLinkToken(ctx, MagicLinkToken{
		UserPublicID: result.PublicID,
		TokenHash:    hash,
		ExpiresAt:    expiresAt,
	}); err != nil {
		return err
	}
	if s.email == nil || s.emailFrom == "" {
		return nil
	}

	link := s.baseURL + "/login/magic-link/" + raw
	html, err := s.email.Render("invite.html", map[string]string{
		"Link":      link,
		"ExpiresAt": expiresAt.Format("02 Jan 2006 15:04 MST"),
	})
	if err != nil {
		return err
	}
	return s.email.Send(ctx, email.Message{
		To:             result.Email,
		From:           s.emailFrom,
		Subject:        i18n.T("auth.invitation_subject"),
		TextBody:       i18n.T("auth.invitation_body", link),
		HTMLBody:       html,
		IdempotencyKey: "invite:" + result.PublicID + ":" + hash,
	})
}

// ConsumeMagicLink verifies and burns a sign-in token, returning the
// authenticated session for the owning account.
func (s *Service) ConsumeMagicLink(ctx context.Context, rawToken string) (SessionResponse, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return SessionResponse{}, ErrInvalidInput
	}
	publicID, err := s.repository.ConsumeMagicLinkToken(ctx, hashMagicToken(rawToken))
	if errors.Is(err, ErrNotFound) {
		return SessionResponse{}, ErrInvalidToken
	}
	if err != nil {
		return SessionResponse{}, err
	}
	session, err := s.Session(ctx, publicID)
	if err != nil {
		return SessionResponse{}, err
	}
	if err := s.repository.TouchLastLogin(ctx, publicID); err != nil {
		return SessionResponse{}, err
	}
	return session, nil
}

// now is overridable in tests; production uses time.Now.
func (s *Service) now() time.Time {
	return time.Now()
}

func newMagicToken() (string, string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", "", err
	}
	raw := base64.RawURLEncoding.EncodeToString(buffer)
	return raw, hashMagicToken(raw), nil
}

func hashMagicToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
