package apitoken

import (
	"context"
	"sort"
	"sync"
	"time"
)

// MemoryRepository is the deterministic in-memory repository used by service
// tests. It is never selected by production wiring.
type MemoryRepository struct {
	mu     sync.RWMutex
	tokens map[string]TokenResult
	now    func() time.Time
}

var _ Repository = (*MemoryRepository)(nil)

// NewMemoryRepository constructs an isolated-test API token store.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		tokens: map[string]TokenResult{},
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// Create stores a new API token.
func (r *MemoryRepository) Create(_ context.Context, input CreateTokenInput) (TokenResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := TokenResult{
		PublicID:     input.PublicID,
		UserPublicID: input.UserPublicID,
		Name:         input.Name,
		TokenHash:    input.TokenHash,
		Prefix:       input.Prefix,
		Scopes:       input.Scopes,
		ExpiresAt:    input.ExpiresAt,
		CreatedAt:    r.now(),
	}
	r.tokens[input.PublicID] = result
	return result, nil
}

// ListByUser returns the active tokens of one user, newest first.
func (r *MemoryRepository) ListByUser(_ context.Context, userPublicID string) ([]TokenResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	results := make([]TokenResult, 0)
	for _, result := range r.tokens {
		if result.UserPublicID == userPublicID && result.RevokedAt == nil {
			results = append(results, result)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.After(results[j].CreatedAt)
	})
	return results[:min(len(results), 100)], nil
}

// FindByPublicID resolves one token by its public identifier.
func (r *MemoryRepository) FindByPublicID(_ context.Context, publicID string) (TokenResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result, ok := r.tokens[publicID]
	if !ok {
		return TokenResult{}, ErrNotFound
	}
	return result, nil
}

// Revoke marks one token owned by the user as revoked.
func (r *MemoryRepository) Revoke(_ context.Context, userPublicID, publicID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	result, ok := r.tokens[publicID]
	if !ok || result.UserPublicID != userPublicID || result.RevokedAt != nil {
		return ErrNotFound
	}
	now := r.now()
	result.RevokedAt = &now
	r.tokens[publicID] = result
	return nil
}

// RevokeAll revokes every active token of one user.
func (r *MemoryRepository) RevokeAll(_ context.Context, userPublicID string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.now()
	revoked := 0
	for publicID, result := range r.tokens {
		if result.UserPublicID != userPublicID || result.RevokedAt != nil {
			continue
		}
		result.RevokedAt = &now
		r.tokens[publicID] = result
		revoked++
	}
	return revoked, nil
}

// TouchLastUsed records the last successful use of a token.
func (r *MemoryRepository) TouchLastUsed(_ context.Context, publicID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	result, ok := r.tokens[publicID]
	if !ok {
		return ErrNotFound
	}
	now := r.now()
	result.LastUsedAt = &now
	r.tokens[publicID] = result
	return nil
}
