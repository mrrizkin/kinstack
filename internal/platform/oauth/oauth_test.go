package oauth

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestStateSignerRoundTrip(t *testing.T) {
	signer, err := NewStateSigner("0123456789abcdef0123456789abcdef", time.Minute)
	if err != nil {
		t.Fatalf("new state signer: %v", err)
	}
	now := time.Now().UTC()
	sealed, err := signer.Seal(StatePayload{
		Purpose:  PurposeOAuth,
		State:    "state-value",
		Provider: "sso",
		Nonce:    "nonce-value",
	}, now)
	if err != nil {
		t.Fatalf("seal state: %v", err)
	}
	payload, err := signer.Open(sealed, now)
	if err != nil {
		t.Fatalf("open state: %v", err)
	}
	if payload.State != "state-value" || payload.Provider != "sso" ||
		payload.Nonce != "nonce-value" || payload.Purpose != PurposeOAuth {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Expires != now.Add(time.Minute).Unix() {
		t.Fatalf("unexpected expiry: %d", payload.Expires)
	}
	if _, err := signer.Open(sealed, now.Add(2*time.Minute)); err == nil {
		t.Fatal("expected expired state to fail")
	}
}

func TestStateSignerRejectsTamperedSeal(t *testing.T) {
	signer, err := NewStateSigner("0123456789abcdef0123456789abcdef", time.Minute)
	if err != nil {
		t.Fatalf("new state signer: %v", err)
	}
	now := time.Now().UTC()
	sealed, err := signer.Seal(StatePayload{State: "s", Provider: "p"}, now)
	if err != nil {
		t.Fatalf("seal state: %v", err)
	}
	// Flip one character of the ciphertext region.
	tampered := []byte(sealed)
	position := len(tampered) - 2
	if tampered[position] == 'A' {
		tampered[position] = 'B'
	} else {
		tampered[position] = 'A'
	}
	if _, err := signer.Open(string(tampered), now); err == nil {
		t.Fatal("expected tampered state to fail")
	}
	if _, err := signer.Open("v1.", now); err == nil {
		t.Fatal("expected empty sealed body to fail")
	}
	if _, err := signer.Open("v2.whatever", now); err == nil {
		t.Fatal("expected unknown version to fail")
	}
}

func TestStateSignerRequiresMatchingKey(t *testing.T) {
	first, err := NewStateSigner("0123456789abcdef0123456789abcdef", time.Minute)
	if err != nil {
		t.Fatalf("new state signer: %v", err)
	}
	second, err := NewStateSigner("fedcba9876543210fedcba9876543210", time.Minute)
	if err != nil {
		t.Fatalf("new state signer: %v", err)
	}
	now := time.Now().UTC()
	sealed, err := first.Seal(StatePayload{State: "s", Provider: "p"}, now)
	if err != nil {
		t.Fatalf("seal state: %v", err)
	}
	if _, err := second.Open(sealed, now); err == nil {
		t.Fatal("expected a different key to reject the state")
	}
}

func TestStateSignerRequiresSecretAndTTL(t *testing.T) {
	if _, err := NewStateSigner("short", time.Minute); err == nil {
		t.Fatal("expected a short secret to be rejected")
	}
	if _, err := NewStateSigner("0123456789abcdef0123456789abcdef", 0); err == nil {
		t.Fatal("expected a non-positive ttl to be rejected")
	}
}

func TestCodeChallengeIsDeterministic(t *testing.T) {
	first := CodeChallenge("verifier")
	if first == "" || first != CodeChallenge("verifier") {
		t.Fatal("expected deterministic non-empty challenge")
	}
	if first == CodeChallenge("other") {
		t.Fatal("expected different verifier to change the challenge")
	}
}

func TestHTTPProviderAuthorizationAndExchange(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"access_token":"token-value","token_type":"Bearer"}`)
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer token-value" {
				http.Error(w, "missing bearer", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"sub":"subject-1","email":"user@example.com","email_verified":true,"name":"User"}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := &HTTPProvider{
		ProviderName: "sso",
		ClientID:     "client",
		ClientSecret: "secret",
		AuthURL:      server.URL + "/authorize",
		TokenURL:     server.URL + "/token",
		UserInfoURL:  server.URL + "/userinfo",
		HTTPClient:   server.Client(),
	}
	redirect, err := provider.Start(context.Background(), AuthorizationRequest{
		RedirectURI:  "https://app.example.com/login/oauth/callback",
		State:        "state-1",
		Nonce:        "nonce-1",
		CodeVerifier: "verifier-1",
	}, nil)
	if err != nil {
		t.Fatalf("authorization url: %v", err)
	}
	parsed, err := url.Parse(redirect)
	if err != nil {
		t.Fatalf("parse authorization url: %v", err)
	}
	query := parsed.Query()
	if query.Get("code_challenge_method") != "S256" || query.Get("code_challenge") != CodeChallenge("verifier-1") {
		t.Fatalf("unexpected PKCE parameters: %s", redirect)
	}

	identity, err := provider.Complete(context.Background(), ExchangeRequest{
		Code:        "code-1",
		Verifier:    "verifier-1",
		Nonce:       "nonce-1",
		State:       "state-1",
		RedirectURI: "https://app.example.com/login/oauth/callback",
	}, nil)
	if err != nil {
		t.Fatalf("exchange: %v", err)
	}
	if identity.Subject != "subject-1" || identity.Email != "user@example.com" ||
		!identity.EmailVerified || identity.DisplayName != "User" {
		t.Fatalf("unexpected identity: %#v", identity)
	}
}
