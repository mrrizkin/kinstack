package auth

import (
	"context"
	"errors"
	"testing"

	platformoauth "kinstack/internal/platform/oauth"
)

func newIdentityTestService() (*Service, *MemoryRepository) {
	repository := NewMemoryRepository()
	repository.AddUser(FindUserByUsernameResult{
		PublicID:    "usr_1",
		Username:    "admin",
		DisplayName: "Administrator",
		Email:       "admin@example.com",
		Role:        testAdminRole,
		IsActive:    true,
	})
	return NewService(ServiceDependency{Repository: repository}), repository
}

func identity(provider, subject, email string, verified bool) platformoauth.Identity {
	return platformoauth.Identity{
		Provider:      provider,
		Subject:       subject,
		Email:         email,
		EmailVerified: verified,
	}
}

func TestOAuthSignInBindsSubjectAndKeepsItStable(t *testing.T) {
	service, _ := newIdentityTestService()
	ctx := context.Background()

	first, err := service.OAuthSignIn(ctx, identity("fake", "subject-1", "admin@example.com", true))
	if err != nil {
		t.Fatalf("first sign-in: %v", err)
	}
	if first.PublicID != "usr_1" {
		t.Fatalf("unexpected account: %+v", first)
	}

	// The provider may stop asserting the email (or change it); the binding is
	// what authenticates the account after the first link.
	second, err := service.OAuthSignIn(ctx, identity("fake", "subject-1", "", false))
	if err != nil {
		t.Fatalf("returning sign-in: %v", err)
	}
	if second.PublicID != first.PublicID {
		t.Fatalf("binding changed account: %q != %q", second.PublicID, first.PublicID)
	}
}

func TestOAuthSignInRequiresVerifiedEmailForFirstLink(t *testing.T) {
	service, _ := newIdentityTestService()
	if _, err := service.OAuthSignIn(
		context.Background(),
		identity("fake", "subject-1", "admin@example.com", false),
	); !errors.Is(err, ErrOAuthEmailUnverified) {
		t.Fatalf("expected ErrOAuthEmailUnverified, got %v", err)
	}
}

func TestOAuthSignInRejectsEmptyProviderIdentity(t *testing.T) {
	service, _ := newIdentityTestService()
	for _, candidate := range []platformoauth.Identity{
		identity("", "subject-1", "admin@example.com", true),
		identity("fake", "  ", "admin@example.com", true),
	} {
		if _, err := service.OAuthSignIn(
			context.Background(),
			candidate,
		); !errors.Is(
			err,
			ErrOAuthUnregistered,
		) {
			t.Fatalf("expected ErrOAuthUnregistered for empty provider/subject, got %v", err)
		}
	}
}

func TestMemoryRepositoryRejectsRebindingSubject(t *testing.T) {
	_, repository := newIdentityTestService()
	ctx := context.Background()

	if err := repository.UpsertIdentity(
		ctx,
		"usr_1",
		"fake",
		"subject-1",
		"admin@example.com",
	); err != nil {
		t.Fatalf("bind: %v", err)
	}
	err := repository.UpsertIdentity(ctx, "usr_2", "fake", "subject-1", "other@example.com")
	if !errors.Is(err, ErrIdentityConflict) {
		t.Fatalf("expected ErrIdentityConflict, got %v", err)
	}

	resolved, err := repository.FindIdentityUser(ctx, "FAKE", " subject-1 ")
	if err != nil {
		t.Fatalf("resolve identity: %v", err)
	}
	if resolved.PublicID != "usr_1" {
		t.Fatalf("identity resolved to %q", resolved.PublicID)
	}
	if _, err := repository.FindIdentityUser(ctx, "fake", "unknown"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
