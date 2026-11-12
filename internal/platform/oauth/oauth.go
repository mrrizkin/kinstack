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
	ErrUnavailable   = errors.New("oauth provider is unavailable")
	ErrInvalidState  = errors.New("oauth state is invalid or expired")
	ErrInvalidCode   = errors.New("oauth authorization code is invalid")
	ErrInvalidConfig = errors.New("oauth provider configuration is invalid")
)

// Identity is the verified external subject returned by a provider.
type Identity struct {
	// Provider and Subject form the local identity binding.
	Provider      string
	Subject       string
	Email         string
	EmailVerified bool
	DisplayName   string
}

type AuthorizationRequest struct {
	RedirectURI  string
	State        string
	Nonce        string
	CodeVerifier string
}

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
	Label() string
	// Start returns the URL to send the browser to; continuation state goes to store.
	Start(context.Context, AuthorizationRequest, ContinuationStore) (string, error)
	Complete(context.Context, ExchangeRequest, ContinuationStore) (Identity, error)
}

// UnavailableProvider is the fail-closed placeholder used when SSO is disabled.
type UnavailableProvider struct{}

func (UnavailableProvider) Name() string { return "unavailable" }

func (UnavailableProvider) Label() string { return "Unavailable" }

func (UnavailableProvider) Start(
	context.Context,
	AuthorizationRequest,
	ContinuationStore,
) (string, error) {
	return "", ErrUnavailable
}

func (UnavailableProvider) Complete(
	context.Context,
	ExchangeRequest,
	ContinuationStore,
) (Identity, error) {
	return Identity{}, ErrUnavailable
}

// HTTPProvider is the generic OAuth 2.0 / OIDC code adapter; endpoints are
// injected at composition time.
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

func (p *HTTPProvider) Complete(
	ctx context.Context,
	exchange ExchangeRequest,
	_ ContinuationStore,
) (Identity, error) {
	if err := p.validate(); err != nil {
		return Identity{}, err
	}
	if err := validateExchange(exchange); err != nil {
		return Identity{}, err
	}
	token, err := p.exchangeCode(ctx, exchange)
	if err != nil {
		return Identity{}, err
	}
	return p.loadIdentity(ctx, token)
}

// validateExchange fails before any network call.
func validateExchange(exchange ExchangeRequest) error {
	if strings.TrimSpace(exchange.Code) == "" || strings.TrimSpace(exchange.Verifier) == "" ||
		strings.TrimSpace(exchange.Nonce) == "" || strings.TrimSpace(exchange.RedirectURI) == "" {
		return ErrInvalidCode
	}
	return nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func (p *HTTPProvider) exchangeCode(
	ctx context.Context,
	exchange ExchangeRequest,
) (tokenResponse, error) {
	form := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {exchange.Code},
		"redirect_uri":  {exchange.RedirectURI},
		"code_verifier": {exchange.Verifier},
	}
	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		p.TokenURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return tokenResponse{}, fmt.Errorf(
			"%w: create oauth token request: %w",
			ErrUnavailable,
			err,
		)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(p.ClientID, p.ClientSecret)

	response, err := p.httpClient().Do(request)
	if err != nil {
		return tokenResponse{}, fmt.Errorf("%w: exchange oauth code: %w", ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return tokenResponse{}, ErrInvalidCode
	}
	var token tokenResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&token); err != nil ||
		token.AccessToken == "" {
		return tokenResponse{}, ErrInvalidCode
	}
	return token, nil
}

func (p *HTTPProvider) loadIdentity(ctx context.Context, token tokenResponse) (Identity, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.UserInfoURL, nil)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: create oauth userinfo request: %w", ErrUnavailable, err)
	}
	tokenType := strings.TrimSpace(token.TokenType)
	if tokenType == "" {
		tokenType = "Bearer"
	}
	request.Header.Set("Authorization", tokenType+" "+token.AccessToken)

	response, err := p.httpClient().Do(request)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: load oauth identity: %w", ErrUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Identity{}, ErrInvalidCode
	}
	var identity struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 64<<10)).
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

func CodeChallenge(verifier string) string {
	digest := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(digest[:])
}

// StateSigner seals short-lived SSO state: AES-256-GCM so it survives the
// redirect without a session cookie.
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
