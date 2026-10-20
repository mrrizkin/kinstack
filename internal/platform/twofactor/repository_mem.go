package twofactor

import (
	"context"
	"sync"
)

// MemoryRepository is the deterministic in-memory two-factor repository used by
// tests. It is never selected by production wiring.
type MemoryRepository struct {
	mu    sync.Mutex
	users map[string]*memoryUser
}

type memoryUser struct {
	enabled bool
	secret  string
	codes   map[string]int64
	nextID  int64
}

var _ Repository = (*MemoryRepository)(nil)

// NewMemoryRepository constructs an isolated-test repository with the supplied
// known user public IDs.
func NewMemoryRepository(userPublicIDs ...string) *MemoryRepository {
	repository := &MemoryRepository{users: map[string]*memoryUser{}}
	for _, publicID := range userPublicIDs {
		repository.users[publicID] = &memoryUser{codes: map[string]int64{}}
	}
	return repository
}

// FindStatus returns the stored state of one account.
func (r *MemoryRepository) FindStatus(ctx context.Context, userPublicID string) (StatusResult, error) {
	if err := ctx.Err(); err != nil {
		return StatusResult{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userPublicID]
	if !ok {
		return StatusResult{}, ErrNotFound
	}
	return StatusResult{Enabled: user.enabled, Secret: user.secret}, nil
}

// SetSecret stores a pending secret.
func (r *MemoryRepository) SetSecret(ctx context.Context, userPublicID, secret string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userPublicID]
	if !ok {
		return ErrNotFound
	}
	user.secret = secret
	user.enabled = false
	return nil
}

// Enable activates two-factor auth.
func (r *MemoryRepository) Enable(ctx context.Context, userPublicID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userPublicID]
	if !ok {
		return ErrNotFound
	}
	user.enabled = true
	return nil
}

// Disable deactivates two-factor auth.
func (r *MemoryRepository) Disable(ctx context.Context, userPublicID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userPublicID]
	if !ok {
		return ErrNotFound
	}
	user.enabled = false
	user.secret = ""
	return nil
}

// ReplaceRecoveryCodes swaps the stored recovery code hashes.
func (r *MemoryRepository) ReplaceRecoveryCodes(
	ctx context.Context,
	userPublicID string,
	hashes []string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userPublicID]
	if !ok {
		return ErrNotFound
	}
	user.codes = map[string]int64{}
	for _, hash := range hashes {
		user.nextID++
		user.codes[hash] = user.nextID
	}
	return nil
}

// FindRecoveryCode returns the id of an unused recovery code.
func (r *MemoryRepository) FindRecoveryCode(
	ctx context.Context,
	userPublicID string,
	hash string,
) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userPublicID]
	if !ok {
		return 0, ErrNotFound
	}
	id, ok := user.codes[hash]
	if !ok {
		return 0, ErrNotFound
	}
	return id, nil
}

// MarkRecoveryCodeUsed consumes a recovery code.
func (r *MemoryRepository) MarkRecoveryCodeUsed(ctx context.Context, id int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, user := range r.users {
		for hash, codeID := range user.codes {
			if codeID == id {
				delete(user.codes, hash)
				return nil
			}
		}
	}
	return ErrNotFound
}
