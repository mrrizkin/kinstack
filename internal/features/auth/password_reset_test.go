package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeRevoker struct {
	revoked []string
	err     error
}

func (r *fakeRevoker) RevokeAll(_ context.Context, userPublicID string) error {
	r.revoked = append(r.revoked, userPublicID)
	return r.err
}

func seededResetService(t *testing.T, revoker CredentialRevoker, rawToken string) (*Service, *MemoryRepository) {
	t.Helper()
	repository := NewMemoryRepository()
	seedUser(t, repository, "rahasia123")
	repository.AddUser(FindUserByUsernameResult{
		PublicID:     "usr_1",
		Username:     "admin",
		DisplayName:  "Administrator",
		Email:        "admin@example.com",
		PasswordHash: repository.users["admin"].PasswordHash,
		Role:         testAdminRole,
		IsActive:     true,
	})
	if err := repository.CreatePasswordResetToken(context.Background(), PasswordResetToken{
		TokenHash:    hashMagicToken(rawToken),
		UserPublicID: "usr_1",
		ExpiresAt:    time.Now().Add(time.Hour),
	}); err != nil {
		t.Fatalf("seed reset token: %v", err)
	}
	return NewService(ServiceDependency{Repository: repository, Revoker: revoker}), repository
}

func TestResetPasswordRevokesSessionsAndTokens(t *testing.T) {
	revoker := &fakeRevoker{}
	service, _ := seededResetService(t, revoker, "reset-token-raw")

	if err := service.ResetPassword(context.Background(), "reset-token-raw", "password-baru-1"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
	if len(revoker.revoked) != 1 || revoker.revoked[0] != "usr_1" {
		t.Fatalf("expected credentials to be revoked, got %#v", revoker.revoked)
	}
	if _, err := service.Login(context.Background(), "admin", "password-baru-1"); err != nil {
		t.Fatalf("expected the new password to work: %v", err)
	}
}

func TestResetPasswordAbortsWhenRevocationFails(t *testing.T) {
	revoker := &fakeRevoker{err: errors.New("valkey unavailable")}
	service, _ := seededResetService(t, revoker, "reset-token-raw")

	err := service.ResetPassword(context.Background(), "reset-token-raw", "password-baru-1")
	if err == nil || !strings.Contains(err.Error(), "revoke") {
		t.Fatalf("expected a revocation error, got %v", err)
	}
	if _, err := service.Login(context.Background(), "admin", "password-baru-1"); err == nil {
		t.Fatal("password must not change when revocation fails")
	}
	if _, err := service.Login(context.Background(), "admin", "rahasia123"); err != nil {
		t.Fatalf("old password should still work after an aborted reset: %v", err)
	}
}

func TestResetPasswordWithoutRevokerStillWorks(t *testing.T) {
	service, _ := seededResetService(t, nil, "reset-token-raw")
	if err := service.ResetPassword(context.Background(), "reset-token-raw", "password-baru-1"); err != nil {
		t.Fatalf("reset password: %v", err)
	}
}
