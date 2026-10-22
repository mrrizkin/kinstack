package media

import (
	"context"
	"sort"
	"sync"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/pagination"
)

// MemoryRepository is the deterministic in-memory repository used by service
// tests. It is never selected by production wiring.
type MemoryRepository struct {
	mu     sync.RWMutex
	assets map[string]Asset
}

var _ Repository = (*MemoryRepository)(nil)

// NewMemoryRepository constructs an isolated-test media store.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{assets: map[string]Asset{}}
}

// Seed inserts one asset into the memory store.
func (r *MemoryRepository) Seed(asset Asset) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.assets[asset.PublicID] = asset
}

// List returns one page of the actor's active assets, newest first.
func (r *MemoryRepository) List(
	_ context.Context,
	ownerPublicID string,
	query pagination.Query,
) ([]Asset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	assets := make([]Asset, 0)
	for _, asset := range r.assets {
		if asset.OwnerPublicID == ownerPublicID {
			assets = append(assets, asset)
		}
	}
	sort.SliceStable(assets, func(i, j int) bool {
		return assets[i].CreatedAt.After(assets[j].CreatedAt)
	})
	start, end := pagination.Window(query, len(assets))
	return assets[start:end], nil
}

// Create persists one uploaded asset.
func (r *MemoryRepository) Create(_ context.Context, input CreateAssetInput) (Asset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	asset := Asset{
		PublicID:      input.PublicID,
		OwnerPublicID: input.OwnerPublicID,
		Filename:      input.Filename,
		ContentType:   input.ContentType,
		SizeBytes:     input.SizeBytes,
		ObjectKey:     input.ObjectKey,
		CreatedAt:     time.Now().UTC(),
	}
	r.assets[asset.PublicID] = asset
	return asset, nil
}

// Count returns the number of assets owned by the actor.
func (r *MemoryRepository) Count(_ context.Context, ownerPublicID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, asset := range r.assets {
		if asset.OwnerPublicID == ownerPublicID {
			count++
		}
	}
	return count, nil
}

// Find returns one active asset owned by the actor.
func (r *MemoryRepository) Find(_ context.Context, ownerPublicID, publicID string) (Asset, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	asset, ok := r.assets[publicID]
	if !ok || asset.OwnerPublicID != ownerPublicID {
		return Asset{}, ErrNotFound
	}
	return asset, nil
}

// Delete soft-deletes one owned asset.
func (r *MemoryRepository) Delete(
	_ context.Context,
	ownerPublicID, publicID string,
) (Asset, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	asset, ok := r.assets[publicID]
	if !ok || asset.OwnerPublicID != ownerPublicID {
		return Asset{}, ErrNotFound
	}
	delete(r.assets, publicID)
	return asset, nil
}

// SeedDefault inserts an asset owned by the given actor.
func SeedDefault(repository *MemoryRepository, ownerPublicID, publicID, filename string) {
	assert.Assert(repository != nil, "memory repository is required")

	repository.Seed(Asset{
		PublicID:      publicID,
		OwnerPublicID: ownerPublicID,
		Filename:      filename,
		ContentType:   "image/png",
		SizeBytes:     1024,
		ObjectKey:     "media/" + ownerPublicID + "/" + publicID + ".png",
		CreatedAt:     time.Now().UTC(),
	})
}
