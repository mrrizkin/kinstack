package apitoken

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"kinstack/internal/pkg/pagination"
)

func newTestService() (*Service, *MemoryRepository) {
	repository := NewMemoryRepository()
	return NewService(ServiceDependency{Repository: repository}), repository
}

func TestCreateReturnsUsableTokenOnce(t *testing.T) {
	service, _ := newTestService()

	created, err := service.Create(
		context.Background(),
		"user-1",
		"CI pipeline",
		0,
		[]string{"users.view"},
	)
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

	page, err := service.List(context.Background(), "user-1", pagination.New(1, 10))
	if err != nil {
		t.Fatalf("list tokens: %v", err)
	}
	if len(page.Tokens) != 1 || page.Tokens[0].PublicID != created.APIToken.PublicID {
		t.Fatalf("unexpected token list: %#v", page.Tokens)
	}
	if page.Page.Total != 1 {
		t.Fatalf("unexpected page: %+v", page.Page)
	}
}

func TestCreateValidatesName(t *testing.T) {
	service, _ := newTestService()
	if _, err := service.Create(
		context.Background(),
		"user-1",
		"",
		0,
		nil,
	); !errors.Is(
		err,
		ErrInvalidInput,
	) {
		t.Fatalf("expected invalid input, got %v", err)
	}
	if _, err := service.Create(
		context.Background(),
		"user-1",
		strings.Repeat("a", maxNameLength+1),
		0,
		nil,
	); !errors.Is(
		err,
		ErrInvalidInput,
	) {
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
		if _, _, err := service.Authenticate(
			context.Background(),
			raw,
		); !errors.Is(
			err,
			ErrInvalidToken,
		) {
			t.Fatalf("token %q: expected invalid token, got %v", raw, err)
		}
	}
}

func TestAuthenticateRejectsUnknownToken(t *testing.T) {
	service, _ := newTestService()
	raw := tokenScheme + "unknownpublicid" + "." + strings.Repeat("a", 43)
	if _, _, err := service.Authenticate(
		context.Background(),
		raw,
	); !errors.Is(
		err,
		ErrInvalidToken,
	) {
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
	if _, _, err := service.Authenticate(
		context.Background(),
		tampered,
	); !errors.Is(
		err,
		ErrInvalidToken,
	) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}

func TestAuthenticateRejectsRevokedToken(t *testing.T) {
	service, _ := newTestService()
	created, err := service.Create(context.Background(), "user-1", "revoke me", 0, nil)
	if err != nil {
		t.Fatalf("create token: %v", err)
	}
	if err := service.Revoke(
		context.Background(),
		"user-1",
		created.APIToken.PublicID,
	); err != nil {
		t.Fatalf("revoke token: %v", err)
	}
	if _, _, err := service.Authenticate(
		context.Background(),
		created.Token,
	); !errors.Is(
		err,
		ErrInvalidToken,
	) {
		t.Fatalf("expected invalid token, got %v", err)
	}
	if err := service.Revoke(
		context.Background(),
		"user-2",
		created.APIToken.PublicID,
	); !errors.Is(
		err,
		ErrNotFound,
	) {
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
	if _, _, err := service.Authenticate(
		context.Background(),
		created.Token,
	); !errors.Is(
		err,
		ErrInvalidToken,
	) {
		t.Fatalf("expected invalid token, got %v", err)
	}
}

func TestRevokeAllRevokesEveryToken(t *testing.T) {
	service, repository := newTestService()
	ctx := context.Background()
	first, err := service.Create(ctx, "user-1", "first", 0, nil)
	if err != nil {
		t.Fatalf("create first token: %v", err)
	}
	second, err := service.Create(ctx, "user-1", "second", 0, nil)
	if err != nil {
		t.Fatalf("create second token: %v", err)
	}
	other, err := service.Create(ctx, "user-2", "other", 0, nil)
	if err != nil {
		t.Fatalf("create other token: %v", err)
	}
	if err := service.RevokeAll(ctx, "user-1"); err != nil {
		t.Fatalf("revoke all: %v", err)
	}
	for _, token := range []string{first.Token, second.Token} {
		if _, _, err := service.Authenticate(ctx, token); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("expected revoked token to fail, got %v", err)
		}
	}
	if _, _, err := service.Authenticate(ctx, other.Token); err != nil {
		t.Fatalf("another user's token must survive: %v", err)
	}
	if repository == nil {
		t.Fatal("expected a repository")
	}
}

func TestListPaginatesActiveTokens(t *testing.T) {
	service, _ := newTestService()
	for index := 1; index <= 7; index++ {
		if _, err := service.Create(
			context.Background(),
			"user-1",
			fmt.Sprintf("token-%d", index),
			0,
			[]string{"users.view"},
		); err != nil {
			t.Fatalf("create token %d: %v", index, err)
		}
	}
	if _, err := service.Create(
		context.Background(),
		"user-2",
		"other user",
		0,
		[]string{"users.view"},
	); err != nil {
		t.Fatalf("create other token: %v", err)
	}

	first, err := service.List(context.Background(), "user-1", pagination.New(1, 5))
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if first.Page.Total != 7 || first.Page.TotalPages != 2 || len(first.Tokens) != 5 {
		t.Fatalf("unexpected first page: %+v (%d tokens)", first.Page, len(first.Tokens))
	}

	second, err := service.List(context.Background(), "user-1", pagination.New(2, 5))
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(second.Tokens) != 2 || second.Page.From != 6 || second.Page.To != 7 {
		t.Fatalf("unexpected second page: %+v (%d tokens)", second.Page, len(second.Tokens))
	}
	for _, token := range append(first.Tokens, second.Tokens...) {
		if token.Name == "other user" {
			t.Fatal("listing leaked another user's token")
		}
	}

	clamped, err := service.List(context.Background(), "user-1", pagination.New(9, 5))
	if err != nil {
		t.Fatalf("list clamped: %v", err)
	}
	if clamped.Page.Page != 2 || len(clamped.Tokens) != 2 {
		t.Fatalf("unexpected clamped page: %+v", clamped.Page)
	}
}
