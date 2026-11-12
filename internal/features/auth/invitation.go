package auth

import (
	"context"
	"strings"
	"time"

	"kinstack/internal/platform/email"
	"kinstack/internal/platform/i18n"
)

// invitationTTL bounds how long an invitation link stays valid. Invitations
// reuse the password reset token store, so redeeming the link sets the first
// password of the invited account.
const invitationTTL = 7 * 24 * time.Hour

// SendInvitation issues a single-use link that lets a freshly invited account
// set its password and sign in for the first time. Unlike a self-service reset
// it reports failures because the caller just created the account.
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

	raw, hash, err := newOpaqueToken()
	if err != nil {
		return err
	}
	expiresAt := s.now().Add(invitationTTL)
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
