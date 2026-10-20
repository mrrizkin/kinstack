package users

import (
	"context"
	"sort"
	"sync"
	"time"

	"kinstack/internal/pkg/assert"
)

// MemoryRepository is the deterministic in-memory repository used by service
// tests. It is never selected by production wiring.
type MemoryRepository struct {
	mu    sync.RWMutex
	users map[string]UserResult
}

var _ Repository = (*MemoryRepository)(nil)

// NewMemoryRepository constructs an empty isolated-test repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{users: make(map[string]UserResult)}
}

// Seed inserts an account projection directly for tests.
func (r *MemoryRepository) Seed(user UserResult) {
	assert.Assert(r != nil, "memory repository is required")
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[user.PublicID] = user
}

func (r *MemoryRepository) ListUsers(_ context.Context) ([]UserResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]UserResult, 0, len(r.users))
	for _, user := range r.users {
		items = append(items, user)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.Before(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryRepository) FindUserByPublicID(
	_ context.Context,
	publicID string,
) (UserResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	user, ok := r.users[publicID]
	if !ok {
		return UserResult{}, ErrNotFound
	}
	return user, nil
}

func (r *MemoryRepository) CreateUser(
	_ context.Context,
	input CreateUserInput,
) (UserResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, user := range r.users {
		if user.Username == input.Username {
			return UserResult{}, ErrUsernameTaken
		}
		if input.Email != "" && user.Email == input.Email {
			return UserResult{}, ErrEmailTaken
		}
	}
	result := UserResult{
		PublicID:    input.PublicID,
		Username:    input.Username,
		Email:       input.Email,
		DisplayName: input.DisplayName,
		IsActive:    true,
		Invited:     input.Invited,
		CreatedAt:   time.Now().UTC(),
	}
	r.users[result.PublicID] = result
	return result, nil
}

func (r *MemoryRepository) UpdateUser(
	_ context.Context,
	input UpdateUserInput,
) (UserResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[input.PublicID]
	if !ok {
		return UserResult{}, ErrNotFound
	}
	for publicID, existing := range r.users {
		if publicID != input.PublicID && input.Email != "" && existing.Email == input.Email {
			return UserResult{}, ErrEmailTaken
		}
	}
	user.Email = input.Email
	user.DisplayName = input.DisplayName
	r.users[user.PublicID] = user
	return user, nil
}

func (r *MemoryRepository) UpdatePassword(_ context.Context, publicID, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.users[publicID]; !ok {
		return ErrNotFound
	}
	return nil
}

func (r *MemoryRepository) SetActive(
	_ context.Context,
	publicID string,
	active bool,
) (UserResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[publicID]
	if !ok {
		return UserResult{}, ErrNotFound
	}
	user.IsActive = active
	r.users[user.PublicID] = user
	return user, nil
}
