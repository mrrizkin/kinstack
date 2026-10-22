package apitoken

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/pagination"
	"kinstack/internal/pkg/publicid"
	"kinstack/internal/platform/authz"
	"kinstack/internal/platform/i18n"

	"github.com/lib/pq"
)

const (
	tokenScheme      = "hus_"
	tokenSecretBytes = 32
	tokenPrefixChars = 6
	minNameLength    = 1
	maxNameLength    = 80
	maxExpiryDays    = 3650
	maxScopes        = 100
	lastUsedThrottle = time.Minute
)

// ServiceDependency contains the persistence boundary and clock used by the
// API token service.
type ServiceDependency struct {
	Repository Repository
	Now        func() time.Time
}

// Service owns API token creation, listing, revocation, and bearer
// authentication.
type Service struct {
	repository Repository
	now        func() time.Time
}

// NewService constructs an API token service around its repository.
func NewService(d ServiceDependency) *Service {
	assert.Assert(d.Repository != nil, "api token repository is required")

	now := d.Now
	if now == nil {
		now = time.Now
	}
	return &Service{repository: d.Repository, now: now}
}

// CreatedToken carries the one-time plaintext token returned at creation.
type CreatedToken struct {
	Token    string
	APIToken APITokenDTO
}

// Create issues a new personal API token and returns its plaintext value once.
func (s *Service) Create(
	ctx context.Context,
	userPublicID string,
	name string,
	expiresInDays int,
	scopes []string,
) (CreatedToken, error) {
	userPublicID = strings.TrimSpace(userPublicID)
	name = strings.TrimSpace(name)
	if userPublicID == "" {
		return CreatedToken{}, ErrInvalidInput
	}
	if len(name) < minNameLength || len(name) > maxNameLength {
		return CreatedToken{}, fmt.Errorf(
			"%w: %s",
			ErrInvalidInput,
			i18n.T("apitoken.name_length", minNameLength, maxNameLength),
		)
	}
	if expiresInDays < 0 || expiresInDays > maxExpiryDays {
		return CreatedToken{}, fmt.Errorf(
			"%w: %s",
			ErrInvalidInput,
			i18n.T("apitoken.expiry_invalid"),
		)
	}
	normalizedScopes, err := normalizeScopes(scopes)
	if err != nil {
		return CreatedToken{}, err
	}

	var expiresAt *time.Time
	if expiresInDays > 0 {
		value := s.now().Add(time.Duration(expiresInDays) * 24 * time.Hour)
		expiresAt = &value
	}

	tokenPublicID := publicid.MustNew()
	secret, err := newSecret()
	if err != nil {
		return CreatedToken{}, err
	}
	result, err := s.repository.Create(ctx, CreateTokenInput{
		PublicID:     tokenPublicID,
		UserPublicID: userPublicID,
		Name:         name,
		TokenHash:    hashSecret(secret),
		Prefix:       tokenScheme + tokenPublicID[:tokenPrefixChars],
		Scopes:       normalizedScopes,
		ExpiresAt:    expiresAt,
	})
	if err != nil {
		return CreatedToken{}, err
	}
	return CreatedToken{
		Token:    tokenScheme + tokenPublicID + "." + secret,
		APIToken: toDTO(result),
	}, nil
}

// List returns every active token of one user, newest first.
func (s *Service) List(
	ctx context.Context,
	userPublicID string,
	query pagination.Query,
) (TokenPage, error) {
	userPublicID = strings.TrimSpace(userPublicID)
	if userPublicID == "" {
		return TokenPage{}, ErrInvalidInput
	}
	total, err := s.repository.CountByUser(ctx, userPublicID)
	if err != nil {
		return TokenPage{}, err
	}
	query = pagination.ClampPage(query, total)
	results, err := s.repository.ListByUser(ctx, userPublicID, query)
	if err != nil {
		return TokenPage{}, err
	}
	tokens := make([]APITokenDTO, 0, len(results))
	for _, result := range results {
		tokens = append(tokens, toDTO(result))
	}
	return TokenPage{Tokens: tokens, Page: pagination.NewPage(query, total)}, nil
}

// Revoke marks one token owned by the user as revoked.
func (s *Service) Revoke(ctx context.Context, userPublicID, tokenPublicID string) error {
	return s.repository.Revoke(
		ctx,
		strings.TrimSpace(userPublicID),
		strings.TrimSpace(tokenPublicID),
	)
}

// RevokeAll revokes every personal API token of one user. Used after a
// password reset so a leaked token cannot keep acting on the account.
func (s *Service) RevokeAll(ctx context.Context, userPublicID string) error {
	userPublicID = strings.TrimSpace(userPublicID)
	if userPublicID == "" {
		return nil
	}
	_, err := s.repository.RevokeAll(ctx, userPublicID)
	return err
}

// Authenticate resolves a raw bearer token to its owning user public ID and
// scope allowlist. The token must exist, be unrevoked, unexpired, and match its
// stored hash. An empty scope list means the token grants nothing.
func (s *Service) Authenticate(ctx context.Context, raw string) (string, []string, error) {
	tokenPublicID, secret, ok := splitToken(raw)
	if !ok {
		return "", nil, ErrInvalidToken
	}
	result, err := s.repository.FindByPublicID(ctx, tokenPublicID)
	if errors.Is(err, ErrNotFound) {
		return "", nil, ErrInvalidToken
	}
	if err != nil {
		return "", nil, err
	}
	if result.RevokedAt != nil {
		return "", nil, ErrInvalidToken
	}
	if result.ExpiresAt != nil && !result.ExpiresAt.After(s.now()) {
		return "", nil, ErrInvalidToken
	}
	if subtle.ConstantTimeCompare([]byte(hashSecret(secret)), []byte(result.TokenHash)) != 1 {
		return "", nil, ErrInvalidToken
	}
	if result.LastUsedAt == nil || s.now().Sub(*result.LastUsedAt) >= lastUsedThrottle {
		if err := s.repository.TouchLastUsed(ctx, tokenPublicID); err != nil {
			return "", nil, err
		}
	}
	scopes := []string(result.Scopes)
	if scopes == nil {
		scopes = []string{}
	}
	return result.UserPublicID, scopes, nil
}

func normalizeScopes(scopes []string) (pq.StringArray, error) {
	seen := make(map[string]struct{}, len(scopes))
	normalized := make(pq.StringArray, 0, len(scopes))
	for _, raw := range scopes {
		code := strings.TrimSpace(raw)
		if code == "" {
			continue
		}
		if !authz.IsKnown(authz.Permission(code)) {
			return nil, fmt.Errorf(
				"%w: %s",
				ErrInvalidInput,
				i18n.T("apitoken.scope_unknown", code),
			)
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		normalized = append(normalized, code)
	}
	if len(normalized) > maxScopes {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, i18n.T("apitoken.max_scopes", maxScopes))
	}
	return normalized, nil
}

func newSecret() (string, error) {
	buf := make([]byte, tokenSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate api token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func splitToken(raw string) (string, string, bool) {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, tokenScheme) {
		return "", "", false
	}
	tokenPublicID, secret, found := strings.Cut(strings.TrimPrefix(raw, tokenScheme), ".")
	if !found || tokenPublicID == "" || secret == "" {
		return "", "", false
	}
	return tokenPublicID, secret, true
}

func toDTO(result TokenResult) APITokenDTO {
	return APITokenDTO{
		PublicID:   result.PublicID,
		Name:       result.Name,
		Prefix:     result.Prefix,
		Scopes:     []string(result.Scopes),
		ExpiresAt:  formatTime(result.ExpiresAt),
		LastUsedAt: formatTime(result.LastUsedAt),
		CreatedAt:  result.CreatedAt.UTC().Format(time.RFC3339),
	}
}

func formatTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
