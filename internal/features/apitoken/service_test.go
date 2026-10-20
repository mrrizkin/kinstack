package apitoken

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func newTestService() (*Service, *MemoryRepository) {
	repository := NewMemoryRepository()
	return NewService(ServiceDependency{Repository: repository}), repository
}

func TestCreateReturnsUsableTokenOnce(t *testing.T) {
	service, _ := newTestService()

	created, err := service.Create(context.Background(), "user-1", "CI pipeline", 0, []string{"users.view"})
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if !strings.HasPrefix(created.Token, tokenScheme) {
		t.Fatalf("token missing scheme: %q", created.Token)
	}
	if created.APIToken.PublicID == "" || created.APIToken.Prefix == "" {
		t.Fatalf("unexpected token metadata: %#v", created.APIToken)
	}

	userPublicID, scopes, err := service.Authenticate(context.Background(), created.Token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if userPublicID != "user-1" {
		t.Fatalf("unexpected owner: %q", userPublicID)
	}
	if len(scopes) != 1 || scopes[0] != "users.view" {
		t.Fatalf("unexpected scopes: %#v", scopes)
	}

	tokens, err := service.List(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("list tokens: %v", err)
	}
	if len(tokens) != 1 || tokens[0].PublicID != created.APIToken.PublicID {
		t.Fatalf("unexpected token list: %#v", tokens)
	}
}

func TestCreateValidatesName(t *testing.T) {
	service, _ := newTestService()
	if _, err := service.Create(context.Background(), "user-1", "", 0, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if _, err := service.Create(context.Background(), "user-1", strings.Repeat("a", maxNameLength+1), 0, nil); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestCreateRejectsUnknownScope(t *testing.T) {
	service, _ := newTestService()
	if _, err := service.Create(
		context.Background(),
		"user-1",
		"bad scope",
		0,
		[]string{"nope.nope"},
	); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestCreateWithoutScopesDeniesEverything(t *testing.T) {
	service, _ := newTestService()
	created, err := service.Create(context.Background(), "user-1", "no scope", 0, nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	_, scopes, err := service.Authenticate(context.Background(), created.Token)
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if scopes == nil || len(scopes) != 0 {
		t.Fatalf("expected a non-nil empty scope list, got %#v", scopes)
	}
}

func TestAuthenticateRejectsMalformedToken(t *testing.T) {
	service, _ := newTestService()
	for _, raw := range []string{"", "nope", "hus_", "hus_onlypublicid", "hus_pub."} {
		if _, _, err := service.Authenticate(context.Background(), raw); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("token %q: expected invalid token, got %v", raw, err)
		}
	}
}

func TestAuthenticateRejectsUnknownToken(t *testing.T) {
	service, _ := newTestService()
	raw := tokenScheme + "unknownpublicid" + "." + strings.Repeat("a", 43)
	if _, _, err := service.Authenticate(context.Background(), raw); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}

func TestAuthenticateRejectsTamperedSecret(t *testing.T) {
	service, _ := newTestService()
	created, err := service.Create(context.Background(), "user-1", "tampered", 0, nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	tampered := created.Token[:len(created.Token)-1] + "x"
	if _, _, err := service.Authenticate(context.Background(), tampered); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}

func TestAuthenticateRejectsRevokedToken(t *testing.T) {
	service, _ := newTestService()
	created, err := service.Create(context.Background(), "user-1", "revoke me", 0, nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := service.Revoke(context.Background(), "user-1", created.APIToken.PublicID); err != nil {
		t.Fatalf("revoke token: %v", err)
	}
	if _, _, err := service.Authenticate(context.Background(), created.Token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token, got %v", err)
	}
	if err := service.Revoke(context.Background(), "user-2", created.APIToken.PublicID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found for another owner, got %v", err)
	}
}

func TestAuthenticateRejectsExpiredToken(t *testing.T) {
	repository := NewMemoryRepository()
	current := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	service := NewService(ServiceDependency{
		Repository: repository,
		Now:        func() time.Time { return current },
	})

	created, err := service.Create(context.Background(), "user-1", "short lived", 1, nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	current = current.Add(48 * time.Hour)
	if _, _, err := service.Authenticate(context.Background(), created.Token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}
