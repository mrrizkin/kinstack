package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"kinstack/internal/platform/email"
	"kinstack/internal/platform/i18n"
	platformpassword "kinstack/internal/platform/password"
)

const (
	// passwordResetTTL bounds how long a self-service reset link stays valid.
	passwordResetTTL = 60 * time.Minute

	minPasswordLength = 8
	maxPasswordLength = 72
)

// RequestPasswordReset emails a single-use reset link to the account owning the
// address. It never reveals whether the address belongs to an account.
func (s *Service) RequestPasswordReset(ctx context.Context, address string) error {
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
	expiresAt := s.now().Add(passwordResetTTL)
	if err := s.repository.CreatePasswordResetToken(ctx, PasswordResetToken{
		UserPublicID: result.PublicID,
		TokenHash:    hash,
		ExpiresAt:    expiresAt,
	}); err != nil {
		return err
	}
	if s.email == nil || s.emailFrom == "" {
		return nil
	}

	link := s.baseURL + "/login/reset/" + raw
	html, err := s.email.Render("password-reset.html", map[string]string{
		"Link":      link,
		"ExpiresAt": expiresAt.Format("02 Jan 2006 15:04 MST"),
	})
	if err != nil {
		return err
	}
	return s.email.Send(ctx, email.Message{
		To:             result.Email,
		From:           s.emailFrom,
		Subject:        i18n.T("auth.reset_subject"),
		TextBody:       i18n.T("auth.reset_body", link),
		HTMLBody:       html,
		IdempotencyKey: "password-reset:" + result.PublicID + ":" + hash,
	})
}

// ResetPassword burns a reset token and stores a new password for its account.
// It returns ErrInvalidToken for unknown, used, or expired tokens.
func (s *Service) ResetPassword(ctx context.Context, rawToken string, newPassword string) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return ErrInvalidInput
	}
	switch {
	case len(newPassword) < minPasswordLength:
		return fmt.Errorf("%w: new password must be at least %d characters", ErrInvalidInput, minPasswordLength)
	case len(newPassword) > maxPasswordLength:
		return fmt.Errorf("%w: new password must be at most %d characters", ErrInvalidInput, maxPasswordLength)
	}

	publicID, err := s.repository.ConsumePasswordResetToken(ctx, hashMagicToken(rawToken))
	if errors.Is(err, ErrNotFound) {
		return ErrInvalidToken
	}
	if err != nil {
		return err
	}

	// Revoke before storing the new hash: a failure here leaves the old
	// password valid but no live sessions, which is the safe side of the
	// trade-off.
	if s.revoker != nil {
		if err := s.revoker.RevokeAll(ctx, publicID); err != nil {
			return fmt.Errorf("revoke credentials: %w", err)
		}
	}

	hash, err := platformpassword.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.repository.UpdatePasswordHash(ctx, publicID, hash)
}
