package profile

import (
	"context"
	"sync"
	"time"

	"kinstack/internal/pkg/assert"
	"kinstack/internal/pkg/pagination"
)

// MemoryRepository is the deterministic in-memory repository used by service
// tests. It is never selected by production wiring.
type MemoryRepository struct {
	mu      sync.RWMutex
	records map[string]memoryRecord
}

type memoryRecord struct {
	profile  ProfileResult
	password string
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{records: map[string]memoryRecord{}}
}

// Seed inserts one account into the memory store.
func (r *MemoryRepository) Seed(result ProfileResult, passwordHash string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[result.PublicID] = memoryRecord{
		profile:  result,
		password: passwordHash,
	}
}

func (r *MemoryRepository) FindProfile(_ context.Context, publicID string) (ProfileResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.records[publicID]
	if !ok {
		return ProfileResult{}, ErrNotFound
	}
	return record.profile, nil
}

func (r *MemoryRepository) FindPasswordHash(_ context.Context, publicID string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.records[publicID]
	if !ok {
		return "", ErrNotFound
	}
	return record.password, nil
}

func (r *MemoryRepository) UpdateDisplayName(
	_ context.Context,
	publicID string,
	displayName string,
) (ProfileResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.records[publicID]
	if !ok {
		return ProfileResult{}, ErrNotFound
	}
	record.profile.DisplayName = displayName
	r.records[publicID] = record
	return record.profile, nil
}

func (r *MemoryRepository) FindPageSize(_ context.Context, publicID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.records[publicID]
	if !ok {
		return 0, ErrNotFound
	}
	return record.profile.PageSize, nil
}

func (r *MemoryRepository) UpdatePreferences(
	_ context.Context,
	publicID string,
	update PreferencesUpdate,
) (PreferencesResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.records[publicID]
	if !ok {
		return PreferencesResult{}, ErrNotFound
	}
	record.profile.ThemePreference = update.ThemePreference
	record.profile.SoundEnabled = update.SoundEnabled
	record.profile.PageSize = update.PageSize
	r.records[publicID] = record
	return PreferencesResult(update), nil
}

// MarkPasswordSet stores a new password hash and clears the invited marker.
func (r *MemoryRepository) MarkPasswordSet(
	_ context.Context,
	publicID string,
	hash string,
) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.records[publicID]
	if !ok {
		return ErrNotFound
	}
	record.password = hash
	record.profile.Invited = false
	r.records[publicID] = record
	return nil
}

// SeedDefault inserts a ready-to-use account.
func SeedDefault(repository *MemoryRepository, publicID, displayName, hash string) {
	assert.Assert(repository != nil, "memory repository is required")

	now := time.Now().UTC()
	repository.Seed(ProfileResult{
		PublicID:        publicID,
		Username:        publicID,
		DisplayName:     displayName,
		Role:            "administrator",
		ThemePreference: "system",
		SoundEnabled:    true,
		PageSize:        pagination.DefaultPerPage,
		LastLoginAt:     &now,
		CreatedAt:       now,
	}, hash)
}
