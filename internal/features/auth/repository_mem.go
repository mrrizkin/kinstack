package auth

import (
	"context"
	"strings"
	"sync"
	"time"

	"kinstack/internal/pkg/assert"
)

// MemoryRepository is the deterministic in-memory repository used by service
// tests. It is never selected by production wiring.
type MemoryRepository struct {
	mu          sync.RWMutex
	users       map[string]FindUserByUsernameResult
	tokens      map[string]memoryMagicToken
	resetTokens map[string]memoryMagicToken
	identities  map[string]memoryIdentity
	touch       []string
}

type memoryIdentity struct {
	userPublicID string
	email        string
}

type memoryMagicToken struct {
	userPublicID string
	expiresAt    time.Time
	used         bool
}

var _ Repository = (*MemoryRepository)(nil)

// NewMemoryRepository constructs an empty isolated-test repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		users:       make(map[string]FindUserByUsernameResult),
		tokens:      make(map[string]memoryMagicToken),
		resetTokens: make(map[string]memoryMagicToken),
		identities:  make(map[string]memoryIdentity),
	}
}

// AddUser seeds a credential projection for tests.
func (r *MemoryRepository) AddUser(user FindUserByUsernameResult) {
	assert.Assert(r != nil, "memory repository is required")
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[strings.ToLower(strings.TrimSpace(user.Username))] = user
}

// Touched reports accounts whose last-login timestamp was updated.
func (r *MemoryRepository) Touched() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]string(nil), r.touch...)
}

// FindIdentityUser resolves the account bound to a provider subject.
func (r *MemoryRepository) FindIdentityUser(
	_ context.Context,
	provider string,
	subject string,
) (FindUserByUsernameResult, error) {
	r.mu.RLock()
	identity, ok := r.identities[identityKey(provider, subject)]
	r.mu.RUnlock()
	if !ok {
		return FindUserByUsernameResult{}, ErrNotFound
	}
	return r.FindUserByPublicID(context.Background(), identity.userPublicID)
}

// UpsertIdentity binds a provider subject to an account.
func (r *MemoryRepository) UpsertIdentity(
	_ context.Context,
	userPublicID string,
	provider string,
	subject string,
	email string,
) error {
	key := identityKey(provider, subject)
	r.mu.Lock()
	defer r.mu.Unlock()
	owner, exists := r.identities[key]
	if exists && owner.userPublicID != userPublicID {
		return ErrIdentityConflict
	}
	r.identities[key] = memoryIdentity{userPublicID: userPublicID, email: email}
	return nil
}

func identityKey(provider, subject string) string {
	return strings.ToLower(strings.TrimSpace(provider)) + "\x00" + strings.TrimSpace(subject)
}

func (r *MemoryRepository) FindUserByUsername(
	_ context.Context,
	username string,
) (FindUserByUsernameResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[strings.ToLower(strings.TrimSpace(username))]
	if !ok {
		return FindUserByUsernameResult{}, ErrNotFound
	}
	return user, nil
}

func (r *MemoryRepository) FindUserByPublicID(
	_ context.Context,
	publicID string,
) (FindUserByUsernameResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		if user.PublicID == publicID {
			return user, nil
		}
	}
	return FindUserByUsernameResult{}, ErrNotFound
}

func (r *MemoryRepository) FindUserByEmail(
	_ context.Context,
	email string,
) (FindUserByUsernameResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	needle := strings.TrimSpace(email)
	for _, user := range r.users {
		if user.Email != "" && strings.EqualFold(user.Email, needle) {
			return user, nil
		}
	}
	return FindUserByUsernameResult{}, ErrNotFound
}

func (r *MemoryRepository) CreateMagicLinkToken(
	_ context.Context,
	token MagicLinkToken,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens[token.TokenHash] = memoryMagicToken{
		userPublicID: token.UserPublicID,
		expiresAt:    token.ExpiresAt,
	}
	return nil
}

func (r *MemoryRepository) ConsumeMagicLinkToken(
	_ context.Context,
	tokenHash string,
) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	token, ok := r.tokens[tokenHash]
	if !ok || token.used || time.Now().After(token.expiresAt) {
		return "", ErrNotFound
	}
	token.used = true
	r.tokens[tokenHash] = token
	return token.userPublicID, nil
}

func (r *MemoryRepository) TouchLastLogin(_ context.Context, publicID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.touch = append(r.touch, publicID)
	return nil
}

func (r *MemoryRepository) CreatePasswordResetToken(
	_ context.Context,
	token PasswordResetToken,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resetTokens[token.TokenHash] = memoryMagicToken{
		userPublicID: token.UserPublicID,
		expiresAt:    token.ExpiresAt,
	}
	return nil
}

func (r *MemoryRepository) ConsumePasswordResetToken(
	_ context.Context,
	tokenHash string,
) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	token, ok := r.resetTokens[tokenHash]
	if !ok || token.used || time.Now().After(token.expiresAt) {
		return "", ErrNotFound
	}
	token.used = true
	r.resetTokens[tokenHash] = token
	return token.userPublicID, nil
}

func (r *MemoryRepository) UpdatePasswordHash(
	_ context.Context,
	publicID string,
	hash string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, user := range r.users {
		if user.PublicID == publicID {
			user.PasswordHash = hash
			r.users[key] = user
			return nil
		}
	}
	return ErrNotFound
}
