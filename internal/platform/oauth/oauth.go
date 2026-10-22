// Package oauth defines the provider-neutral OAuth 2.0 / OpenID Connect
// boundary used by the auth feature. Provider credentials never cross into the
// auth service: the HTTP adapter owns discovery, tokens, and client secrets.
package oauth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrUnavailable identifies an unreachable or misconfigured provider.
	ErrUnavailable = errors.New("oauth provider is unavailable")
	// ErrInvalidState identifies a forged or expired authorization state.
	ErrInvalidState = errors.New("oauth state is invalid or expired")
	// ErrInvalidCode identifies a rejected authorization code or token response.
	ErrInvalidCode = errors.New("oauth authorization code is invalid")
	// ErrInvalidConfig identifies an incomplete provider configuration.
	ErrInvalidConfig = errors.New("oauth provider configuration is invalid")
)

// Identity is the verified external subject returned by a provider.
type Identity struct {
	// Provider names the identity source (for example "google" or "saml"). It is
	// part of the local identity binding together with Subject.
	Provider      string
	Subject       string
	Email         string
	EmailVerified bool
	DisplayName   string
}

// AuthorizationRequest carries the parameters needed to build a provider URL.
type AuthorizationRequest struct {
	RedirectURI  string
	State        string
	Nonce        string
	CodeVerifier string
}

// ExchangeRequest carries the parameters needed to redeem an authorization code.
type ExchangeRequest struct {
	Code        string
	Verifier    string
	Nonce       string
	State       string
	RedirectURI string
	// Query carries the provider callback parameters (query string or form
	// body). Session-based providers such as goth read code and state from it.
	Query url.Values
}

// ContinuationStore persists short-lived provider state (such as a marshaled
// goth session) between the authorization redirect and the callback. The auth
// handler backs it with the Fiber session.
type ContinuationStore interface {
	Save(key, value string) error
	Load(key string) (string, bool)
	Delete(key string)
}

// Provider is the provider-neutral sign-in boundary consumed by auth.
type Provider interface {
	// Name is the URL-safe slug used in routes and the signed state.
	Name() string
	// Label is the human-readable button label shown on the sign-in page.
	Label() string
	// Start begins the authorization flow and returns the provider URL to send
	// the browser to. Continuation state is stored through store.
	Start(context.Context, AuthorizationRequest, ContinuationStore) (string, error)
	// Complete redeems the callback and returns the verified identity.
	Complete(context.Context, ExchangeRequest, ContinuationStore) (Identity, error)
}

// UnavailableProvider is the fail-closed placeholder used when SSO is disabled.
type UnavailableProvider struct{}

func (UnavailableProvider) Name() string { return "unavailable" }

func (UnavailableProvider) Label() string { return "Unavailable" }

// Start always fails because SSO is disabled.
func (UnavailableProvider) Start(
	context.Context,
	AuthorizationRequest,
	ContinuationStore,
) (string, error) {
	return "", ErrUnavailable
}

// Complete always fails because SSO is disabled.
func (UnavailableProvider) Complete(
	context.Context,
	ExchangeRequest,
	ContinuationStore,
) (Identity, error) {
	return Identity{}, ErrUnavailable
}

// HTTPProvider is a generic OAuth 2.0 / OIDC authorization-code adapter with
// PKCE. The provider-specific endpoints are injected at composition time.
type HTTPProvider struct {
	ProviderName string
	DisplayLabel string
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	HTTPClient   *http.Client
}

func (p *HTTPProvider) Name() string {
	if p == nil || p.ProviderName == "" {
		return "sso"
	}
	return p.ProviderName
}

func (p *HTTPProvider) Label() string {
	if p == nil || p.DisplayLabel == "" {
		return p.Name()
	}
	return p.DisplayLabel
}

func (p *HTTPProvider) validate() error {
	if p == nil || strings.TrimSpace(p.ClientID) == "" || strings.TrimSpace(p.ClientSecret) == "" ||
		strings.TrimSpace(p.AuthURL) == "" || strings.TrimSpace(p.TokenURL) == "" ||
		strings.TrimSpace(p.UserInfoURL) == "" {
		return ErrInvalidConfig
	}
	for _, raw := range []string{p.AuthURL, p.TokenURL, p.UserInfoURL} {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return ErrInvalidConfig
		}
	}
	return nil
}

// Start builds the provider redirect with PKCE and OIDC nonce.
func (p *HTTPProvider) Start(
	_ context.Context,
	request AuthorizationRequest,
	_ ContinuationStore,
) (string, error) {
	if err := p.validate(); err != nil {
		return "", err
	}
	parsed, err := url.Parse(p.AuthURL)
	if err != nil {
		return "", ErrInvalidConfig
	}
	query := parsed.Query()
	query.Set("response_type", "code")
	query.Set("client_id", p.ClientID)
	query.Set("redirect_uri", request.RedirectURI)
	query.Set("scope", "openid email profile")
	query.Set("state", request.State)
	query.Set("nonce", request.Nonce)
	query.Set("code_challenge", CodeChallenge(request.CodeVerifier))
	query.Set("code_challenge_method", "S256")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

// Complete redeems the authorization code and loads the verified identity.
func (p *HTTPProvider) Complete(
	ctx context.Context,
	exchange ExchangeRequest,
	_ ContinuationStore,
) (Identity, error) {
	code, verifier := exchange.Code, exchange.Verifier
	if err := p.validate(); err != nil {
		return Identity{}, err
	}
	if strings.TrimSpace(code) == "" || strings.TrimSpace(verifier) == "" ||
		strings.TrimSpace(exchange.Nonce) == "" || strings.TrimSpace(exchange.RedirectURI) == "" {
		return Identity{}, ErrInvalidCode
	}
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {exchange.RedirectURI},
		"code_verifier": {verifier},
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.TokenURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: create oauth token request: %v", ErrUnavailable, err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(p.ClientID, p.ClientSecret)

	client := p.httpClient()
	response, err := client.Do(request)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: exchange oauth code: %v", ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Identity{}, ErrInvalidCode
	}
	var token struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&token); err != nil ||
		token.AccessToken == "" {
		return Identity{}, ErrInvalidCode
	}
	userRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, p.UserInfoURL, nil)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: create oauth userinfo request: %v", ErrUnavailable, err)
	}
	tokenType := strings.TrimSpace(token.TokenType)
	if tokenType == "" {
		tokenType = "Bearer"
	}
	userRequest.Header.Set("Authorization", tokenType+" "+token.AccessToken)
	userResponse, err := client.Do(userRequest)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: load oauth identity: %v", ErrUnavailable, err)
	}
	defer userResponse.Body.Close()
	if userResponse.StatusCode < 200 || userResponse.StatusCode >= 300 {
		return Identity{}, ErrInvalidCode
	}
	var identity struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(userResponse.Body, 64<<10)).
		Decode(&identity); err != nil {
		return Identity{}, ErrInvalidCode
	}
	return Identity{
		Subject:       strings.TrimSpace(identity.Subject),
		Email:         strings.TrimSpace(identity.Email),
		EmailVerified: identity.EmailVerified,
		DisplayName:   strings.TrimSpace(identity.Name),
	}, nil
}

func (p *HTTPProvider) httpClient() *http.Client {
	if p != nil && p.HTTPClient != nil {
		client := *p.HTTPClient
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
		return &client
	}
	return &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   10 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// RandomToken returns a cryptographically secure URL-safe token.
func RandomToken(size int) (string, error) {
	if size < 16 {
		size = 32
	}
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate oauth token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

// CodeChallenge derives an S256 PKCE challenge from a verifier.
func CodeChallenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

// StateSigner seals and opens short-lived SSO state payloads. The value is
// authenticated and encrypted with AES-256-GCM so it can travel through the
// authorization request (and be posted back by providers that use form_post)
// without depending on a browser session cookie.
type StateSigner struct {
	aead cipher.AEAD
	ttl  time.Duration
}

// SSO state purposes. They keep an OAuth state from being replayed as a SAML
// relay state and vice versa.
const (
	PurposeOAuth = "oauth"
	PurposeSAML  = "saml"
)

// StatePayload is the data carried inside a sealed state value. Keep it small:
// it travels in a URL or in a form field.
type StatePayload struct {
	Purpose  string `json:"purpose"`
	State    string `json:"state"`
	Provider string `json:"provider"`
	Nonce    string `json:"nonce,omitempty"`
	Expires  int64  `json:"expires"`
}

const stateSealedVersion = "v1"

func NewStateSigner(secret string, ttl time.Duration) (*StateSigner, error) {
	if len([]byte(secret)) < 32 || ttl <= 0 {
		return nil, ErrInvalidConfig
	}
	// Derive the AES key from the configured secret with HKDF-style separation
	// so the state key can never be confused with another use of the secret.
	derive := hmac.New(sha256.New, []byte(secret))
	_, _ = derive.Write([]byte("kinstack/sso-state-seal/v1"))
	block, err := aes.NewCipher(derive.Sum(nil))
	if err != nil {
		return nil, ErrInvalidConfig
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, ErrInvalidConfig
	}
	return &StateSigner{aead: aead, ttl: ttl}, nil
}

// Seal returns an opaque, authenticated value carrying the payload.
func (s *StateSigner) Seal(payload StatePayload, now time.Time) (string, error) {
	if s == nil || s.aead == nil ||
		strings.TrimSpace(payload.State) == "" ||
		strings.TrimSpace(payload.Provider) == "" {
		return "", ErrInvalidState
	}
	if payload.Expires == 0 {
		payload.Expires = now.Add(s.ttl).Unix()
	}
	plain, err := json.Marshal(payload)
	if err != nil {
		return "", ErrInvalidState
	}
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", ErrInvalidState
	}
	sealed := s.aead.Seal(nonce, nonce, plain, nil)
	return stateSealedVersion + "." + base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Open authenticates and decrypts a sealed state value. It rejects values with
// the wrong version, a broken tag, an unknown shape, or an expired payload.
func (s *StateSigner) Open(value string, now time.Time) (StatePayload, error) {
	version, encoded, ok := strings.Cut(strings.TrimSpace(value), ".")
	if s == nil || s.aead == nil || !ok || version != stateSealedVersion {
		return StatePayload{}, ErrInvalidState
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil || len(raw) <= s.aead.NonceSize() {
		return StatePayload{}, ErrInvalidState
	}
	size := s.aead.NonceSize()
	plain, err := s.aead.Open(nil, raw[:size], raw[size:], nil)
	if err != nil {
		return StatePayload{}, ErrInvalidState
	}
	var payload StatePayload
	if err := json.Unmarshal(plain, &payload); err != nil ||
		strings.TrimSpace(payload.State) == "" ||
		strings.TrimSpace(payload.Provider) == "" ||
		payload.Expires <= now.Unix() {
		return StatePayload{}, ErrInvalidState
	}
	return payload, nil
}
