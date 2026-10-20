package actor

import (
	"context"
	"strings"
	"sync"
)

// MemoryRepository is an in-memory actor repository for tests and local runs.
type MemoryRepository struct {
	mu     sync.RWMutex
	active map[string]struct{}
}

var _ Repository = (*MemoryRepository)(nil)

// NewMemoryRepository creates a repository with the supplied active actors.
func NewMemoryRepository(publicIDs ...string) *MemoryRepository {
	repository := &MemoryRepository{active: make(map[string]struct{}, len(publicIDs))}
	for _, publicID := range publicIDs {
		repository.Add(publicID)
	}
	return repository
}

// NewMemoryActor creates an Actor backed by an in-memory repository that
// accepts exactly the supplied publicIDs. Intended for tests and local runs.
func NewMemoryActor(publicIDs ...string) *Actor {
	return New(Dependency{Repository: NewMemoryRepository(publicIDs...)})
}

// Add marks an actor as active.
func (r *MemoryRepository) Add(publicID string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active[strings.TrimSpace(publicID)] = struct{}{}
}

// Remove marks an actor as unavailable.
func (r *MemoryRepository) Remove(publicID string) {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.active, strings.TrimSpace(publicID))
}

// FindByPublicID resolves only active actors.
func (r *MemoryRepository) FindByPublicID(ctx context.Context, publicID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r == nil {
		return ErrRepositoryUnavailable
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if _, ok := r.active[strings.TrimSpace(publicID)]; !ok {
		return ErrNotFound
	}
	return nil
}
