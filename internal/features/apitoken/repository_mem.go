package apitoken

import (
	"context"
	"sort"
	"sync"
	"time"

	"kinstack/internal/pkg/pagination"
)

// MemoryRepository is the deterministic in-memory repository used by service
// tests. It is never selected by production wiring.
type MemoryRepository struct {
	mu     sync.RWMutex
	tokens map[string]TokenResult
	now    func() time.Time
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		tokens: map[string]TokenResult{},
		now:    func() time.Time { return time.Now().UTC() },
	}
}

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

func (r *MemoryRepository) ListByUser(
	_ context.Context,
	userPublicID string,
	query pagination.Query,
) ([]TokenResult, error) {
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
	start, end := pagination.Window(query, len(results))
	return results[start:end], nil
}

func (r *MemoryRepository) CountByUser(_ context.Context, userPublicID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, result := range r.tokens {
		if result.UserPublicID == userPublicID && result.RevokedAt == nil {
			count++
		}
	}
	return count, nil
}

func (r *MemoryRepository) FindByPublicID(_ context.Context, publicID string) (TokenResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result, ok := r.tokens[publicID]
	if !ok {
		return TokenResult{}, ErrNotFound
	}
	return result, nil
}

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
