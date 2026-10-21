package profile

import (
	"context"
	"errors"
	"testing"
	"time"

	"kinstack/internal/platform/password"
)

func newService(t *testing.T) (*Service, *MemoryRepository, string) {
	t.Helper()
	repository := NewMemoryRepository()
	hash, err := password.Hash("rahasia12345")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	SeedDefault(repository, "user-1", "Pemilik Contoh", hash)
	return NewService(ServiceDependency{Repository: repository}), repository, "user-1"
}

func TestShowReturnsProfileAndPreferences(t *testing.T) {
	service, _, publicID := newService(t)

	props, err := service.Show(context.Background(), publicID)
	if err != nil {
		t.Fatalf("show profile: %v", err)
	}
	if props.Profile.Username != publicID || props.Profile.DisplayName != "Pemilik Contoh" {
		t.Fatalf("unexpected profile: %#v", props.Profile)
	}
	if props.Preferences.Theme != "system" || !props.Preferences.SoundEnabled {
		t.Fatalf("unexpected preferences: %#v", props.Preferences)
	}
	if props.Profile.LastLoginAt == "" || props.Profile.CreatedAt == "" {
		t.Fatalf("expected timestamps, got %#v", props.Profile)
	}
}

func TestShowRejectsUnknownUser(t *testing.T) {
	service, _, _ := newService(t)

	if _, err := service.Show(context.Background(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateDisplayNameValidatesLength(t *testing.T) {
	service, _, publicID := newService(t)

	if _, err := service.UpdateDisplayName(
		context.Background(),
		publicID,
		UpdateProfileRequest{DisplayName: "ab"},
	); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for short name, got %v", err)
	}

	updated, err := service.UpdateDisplayName(
		context.Background(),
		publicID,
		UpdateProfileRequest{DisplayName: "  Kasir Sore  "},
	)
	if err != nil {
		t.Fatalf("update display name: %v", err)
	}
	if updated.DisplayName != "Kasir Sore" {
		t.Fatalf("expected trimmed name, got %q", updated.DisplayName)
	}
}

func TestUpdatePreferencesValidatesTheme(t *testing.T) {
	service, _, publicID := newService(t)

	if _, err := service.UpdatePreferences(
		context.Background(),
		publicID,
		UpdatePreferencesRequest{Theme: "neon"},
	); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for unknown theme, got %v", err)
	}

	saved, err := service.UpdatePreferences(
		context.Background(),
		publicID,
		UpdatePreferencesRequest{Theme: "Dark", SoundEnabled: false},
	)
	if err != nil {
		t.Fatalf("update preferences: %v", err)
	}
	if saved.Theme != "dark" || saved.SoundEnabled {
		t.Fatalf("unexpected saved preferences: %#v", saved)
	}
}

func TestUpdateThemeKeepsSoundPreference(t *testing.T) {
	service, _, publicID := newService(t)

	if _, err := service.UpdatePreferences(
		context.Background(),
		publicID,
		UpdatePreferencesRequest{Theme: "light", SoundEnabled: false},
	); err != nil {
		t.Fatalf("seed preferences: %v", err)
	}

	saved, err := service.UpdateTheme(context.Background(), publicID, "Dark")
	if err != nil {
		t.Fatalf("update theme: %v", err)
	}
	if saved.Theme != "dark" || saved.SoundEnabled {
		t.Fatalf("unexpected saved preferences: %#v", saved)
	}

	if _, err := service.UpdateTheme(
		context.Background(),
		publicID,
		"neon",
	); !errors.Is(
		err,
		ErrInvalidInput,
	) {
		t.Fatalf("expected ErrInvalidInput for unknown theme, got %v", err)
	}
}

func TestChangePasswordRules(t *testing.T) {
	service, repository, publicID := newService(t)

	if err := service.ChangePassword(context.Background(), publicID, ChangePasswordRequest{
		CurrentPassword: "rahasia12345",
		NewPassword:     "pendek",
		ConfirmPassword: "pendek",
	}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for short password, got %v", err)
	}

	if err := service.ChangePassword(context.Background(), publicID, ChangePasswordRequest{
		CurrentPassword: "rahasia12345",
		NewPassword:     "rahasia67890",
		ConfirmPassword: "berbeda",
	}); !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("expected ErrPasswordMismatch, got %v", err)
	}

	if err := service.ChangePassword(context.Background(), publicID, ChangePasswordRequest{
		CurrentPassword: "salah",
		NewPassword:     "rahasia67890",
		ConfirmPassword: "rahasia67890",
	}); !errors.Is(err, ErrWrongPassword) {
		t.Fatalf("expected ErrWrongPassword, got %v", err)
	}

	if err := service.ChangePassword(context.Background(), publicID, ChangePasswordRequest{
		CurrentPassword: "rahasia12345",
		NewPassword:     "rahasia12345",
		ConfirmPassword: "rahasia12345",
	}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for unchanged password, got %v", err)
	}

	if err := service.ChangePassword(context.Background(), publicID, ChangePasswordRequest{
		CurrentPassword: "rahasia12345",
		NewPassword:     "rahasia67890",
		ConfirmPassword: "rahasia67890",
	}); err != nil {
		t.Fatalf("change password: %v", err)
	}

	hash, err := repository.FindPasswordHash(context.Background(), publicID)
	if err != nil {
		t.Fatalf("find password hash: %v", err)
	}
	if err := password.Compare(hash, "rahasia67890"); err != nil {
		t.Fatalf("expected stored hash to match the new password: %v", err)
	}
	if err := password.Compare(hash, "rahasia12345"); err == nil {
		t.Fatalf("expected old password to stop working")
	}
}

func TestChangePasswordAllowsInvitedAccountWithoutCurrentPassword(t *testing.T) {
	repository := NewMemoryRepository()
	repository.Seed(ProfileResult{
		PublicID:        "invited-1",
		Username:        "invited",
		DisplayName:     "Akun Undangan",
		Role:            "administrator",
		Invited:         true,
		ThemePreference: "system",
		SoundEnabled:    true,
		CreatedAt:       time.Now().UTC(),
	}, "unusable-hash")
	service := NewService(ServiceDependency{Repository: repository})

	if err := service.ChangePassword(context.Background(), "invited-1", ChangePasswordRequest{
		NewPassword:     "rahasia12345",
		ConfirmPassword: "rahasia12345",
	}); err != nil {
		t.Fatalf("change password: %v", err)
	}

	profile, err := repository.FindProfile(context.Background(), "invited-1")
	if err != nil {
		t.Fatalf("find profile: %v", err)
	}
	if profile.Invited {
		t.Fatalf("expected invited flag to be cleared")
	}
	hash, err := repository.FindPasswordHash(context.Background(), "invited-1")
	if err != nil {
		t.Fatalf("find password hash: %v", err)
	}
	if err := password.Compare(hash, "rahasia12345"); err != nil {
		t.Fatalf("expected stored hash to match the new password: %v", err)
	}
}
