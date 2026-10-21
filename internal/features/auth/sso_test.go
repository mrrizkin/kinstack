package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	applog "kinstack/internal/pkg/log"
	"kinstack/internal/platform/actor"
	platformoauth "kinstack/internal/platform/oauth"
	platformsession "kinstack/internal/platform/session"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/storage/memory/v2"
)

// fakeSSOProvider mimics a provider that requires server-side continuation data
// at the callback (PKCE verifier plus a provider session blob).
type fakeSSOProvider struct {
	completed platformoauth.ExchangeRequest
	calls     int
}

func (p *fakeSSOProvider) Name() string  { return "fake" }
func (p *fakeSSOProvider) Label() string { return "Fake SSO" }

func (p *fakeSSOProvider) Start(
	_ context.Context,
	request platformoauth.AuthorizationRequest,
	store platformoauth.ContinuationStore,
) (string, error) {
	if request.CodeVerifier == "" {
		return "", platformoauth.ErrInvalidConfig
	}
	if err := store.Save(request.State, "session-blob"); err != nil {
		return "", err
	}
	return "https://idp.example/authorize?state=" + url.QueryEscape(request.State), nil
}

func (p *fakeSSOProvider) Complete(
	_ context.Context,
	request platformoauth.ExchangeRequest,
	store platformoauth.ContinuationStore,
) (platformoauth.Identity, error) {
	p.calls++
	blob, ok := store.Load(request.State)
	if !ok || blob == "" || request.Verifier == "" || request.Code == "" {
		return platformoauth.Identity{}, platformoauth.ErrInvalidState
	}
	store.Delete(request.State)
	p.completed = request
	return platformoauth.Identity{
		Provider:      "fake",
		Subject:       "subject-1",
		Email:         "admin@example.com",
		EmailVerified: true,
		DisplayName:   "Administrator",
	}, nil
}

func newSSOTestApp(t *testing.T) (*fiber.App, *fakeSSOProvider, *MemoryRepository) {
	t.Helper()
	repository := NewMemoryRepository()
	repository.AddUser(FindUserByUsernameResult{
		PublicID:    "usr_1",
		Username:    "admin",
		DisplayName: "Administrator",
		Email:       "admin@example.com",
		Role:        testAdminRole,
		IsActive:    true,
	})
	provider := &fakeSSOProvider{}
	handler := NewHandler(HandlerDependency{
		Service: NewService(ServiceDependency{Repository: repository}),
		Actor: actor.New(
			actor.Dependency{Repository: actor.NewMemoryRepository("usr_1")},
		),
		Logger:              applog.NewNoopLogger(),
		OAuthProviders:      []platformoauth.Provider{provider},
		OAuthStateSecret:    strings.Repeat("s", 32),
		OAuthRedirectURL:    "https://app.example/login/oauth/callback",
		ContinuationStorage: memory.New(),
	})

	app := fiber.New()
	app.Use(platformsession.NewMiddleware(
		platformsession.NewMemoryStore(30*time.Minute),
		30*time.Minute,
	))
	handler.Mount(app)
	return app, provider, repository
}

// startState performs the authorization redirect and returns the sealed state
// that the provider would echo back.
func startState(t *testing.T, app *fiber.App) string {
	t.Helper()
	response, err := app.Test(httptest.NewRequest(
		http.MethodGet,
		"/login/oauth/start?provider=fake",
		nil,
	))
	if err != nil {
		t.Fatalf("start request: %v", err)
	}
	if response.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("expected 303 from start, got %d", response.StatusCode)
	}
	location, err := url.Parse(response.Header.Get("Location"))
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	state := location.Query().Get("state")
	if state == "" {
		t.Fatal("expected a sealed state in the redirect")
	}
	return state
}

// A cross-site POST callback must not depend on the session cookie: providers
// such as Apple post the authorization response back without one.
func TestOAuthPostCallbackWorksWithoutSessionCookie(t *testing.T) {
	app, provider, _ := newSSOTestApp(t)
	state := startState(t, app)

	form := url.Values{"state": {state}, "code": {"auth-code"}}
	request := httptest.NewRequest(
		http.MethodPost,
		"/login/oauth/callback",
		strings.NewReader(form.Encode()),
	)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Deliberately no Cookie header: the browser would not send a SameSite=Lax
	// cookie with a cross-site POST.
	response, err := app.Test(request)
	if err != nil {
		t.Fatalf("callback request: %v", err)
	}
	if response.StatusCode != fiber.StatusSeeOther {
		t.Fatalf("expected 303 from the callback, got %d", response.StatusCode)
	}
	if provider.calls != 1 {
		t.Fatalf("expected one provider completion, got %d", provider.calls)
	}
	if provider.completed.Verifier == "" {
		t.Fatal("expected the PKCE verifier to survive the stateless callback")
	}
	if provider.completed.Nonce == "" {
		t.Fatal("expected the nonce to survive the sealed state")
	}
	if provider.completed.State != state {
		t.Fatal("expected the callback to use the sealed state")
	}
	if len(response.Header.Values("Set-Cookie")) == 0 {
		t.Fatal("expected the callback to establish a new session cookie")
	}
}

func TestOAuthCallbackRejectsReplayAndTampering(t *testing.T) {
	app, provider, _ := newSSOTestApp(t)
	state := startState(t, app)

	post := func(value string) *http.Response {
		t.Helper()
		form := url.Values{"state": {value}, "code": {"auth-code"}}
		request := httptest.NewRequest(
			http.MethodPost,
			"/login/oauth/callback",
			strings.NewReader(form.Encode()),
		)
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response, err := app.Test(request)
		if err != nil {
			t.Fatalf("callback request: %v", err)
		}
		return response
	}
	signedIn := func(response *http.Response) bool {
		return response.StatusCode == fiber.StatusSeeOther &&
			response.Header.Get("Location") == "/dashboard"
	}

	if response := post(state); !signedIn(response) {
		t.Fatalf(
			"first callback should sign in, got %d %q",
			response.StatusCode,
			response.Header.Get("Location"),
		)
	}
	// The transaction data is single use even though the state is stateless.
	if response := post(state); signedIn(response) {
		t.Fatal("replayed state must not sign in again")
	}
	if provider.calls != 2 {
		t.Fatalf("expected the provider to be asked twice, got %d", provider.calls)
	}

	tampered := []byte(state)
	middle := len(tampered) / 2
	if tampered[middle] == 'A' {
		tampered[middle] = 'B'
	} else {
		tampered[middle] = 'A'
	}
	if response := post(string(tampered)); signedIn(response) {
		t.Fatal("tampered state must not sign in")
	}
}

func TestSSOStoreIsNamespacedAndSingleUse(t *testing.T) {
	storage := memory.New()
	oauth := newSSOStore(storage, ssoOAuthPrefix)
	saml := newSSOStore(storage, ssoSAMLRequestID)

	if err := oauth.Save("state-1", "value-1"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, ok := saml.Load("state-1"); ok {
		t.Fatal("saml store must not see oauth entries")
	}
	value, ok := oauth.Consume("state-1")
	if !ok || value != "value-1" {
		t.Fatalf("unexpected consume result: %q %v", value, ok)
	}
	if _, ok := oauth.Load("state-1"); ok {
		t.Fatal("consume must remove the entry")
	}
	if err := newSSOStore(nil, "").Save("state", "value"); err == nil {
		t.Fatal("expected a missing storage to fail closed")
	}
}
