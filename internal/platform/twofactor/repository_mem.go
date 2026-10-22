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
	enabled    bool
	secret     string
	sealed     []byte
	keyVersion int
	lastStep   int64
	codes      map[string]int64
	nextID     int64
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository(userPublicIDs ...string) *MemoryRepository {
	repository := &MemoryRepository{users: map[string]*memoryUser{}}
	for _, publicID := range userPublicIDs {
		repository.users[publicID] = &memoryUser{codes: map[string]int64{}}
	}
	return repository
}

// FindStatus returns the stored state of one account.
func (r *MemoryRepository) FindStatus(
	ctx context.Context,
	userPublicID string,
) (StatusResult, error) {
	if err := ctx.Err(); err != nil {
		return StatusResult{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userPublicID]
	if !ok {
		return StatusResult{}, ErrNotFound
	}
	return StatusResult{
		Enabled:    user.enabled,
		Secret:     user.secret,
		Sealed:     append([]byte(nil), user.sealed...),
		KeyVersion: user.keyVersion,
		LastStep:   user.lastStep,
	}, nil
}

// SetSecret stores a pending secret.
func (r *MemoryRepository) SetSecret(
	ctx context.Context,
	userPublicID string,
	secret Secret,
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
	user.secret = secret.Plain
	user.sealed = append([]byte(nil), secret.Sealed...)
	user.keyVersion = secret.KeyVersion
	user.lastStep = 0
	user.enabled = false
	return nil
}

// SealSecret stores an encrypted copy and drops the plaintext.
func (r *MemoryRepository) SealSecret(
	ctx context.Context,
	userPublicID string,
	sealed []byte,
	version int,
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
	user.sealed = append([]byte(nil), sealed...)
	user.keyVersion = version
	user.secret = ""
	return nil
}

// Enable activates two-factor auth and records the accepted TOTP step.
func (r *MemoryRepository) Enable(ctx context.Context, userPublicID string, lastStep int64) error {
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
	user.lastStep = lastStep
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
	user.sealed = nil
	user.keyVersion = 0
	user.lastStep = 0
	return nil
}

// ConsumeStep records the accepted TOTP step, failing on replay.
func (r *MemoryRepository) ConsumeStep(ctx context.Context, userPublicID string, step int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userPublicID]
	if !ok {
		return ErrNotFound
	}
	if step <= user.lastStep {
		return ErrNotFound
	}
	user.lastStep = step
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

// UnusedRecoveryCodes lists the stored hashes for one account.
func (r *MemoryRepository) UnusedRecoveryCodes(
	ctx context.Context,
	userPublicID string,
) ([]RecoveryCode, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	user, ok := r.users[userPublicID]
	if !ok {
		return nil, ErrNotFound
	}
	codes := make([]RecoveryCode, 0, len(user.codes))
	for hash, id := range user.codes {
		codes = append(codes, RecoveryCode{ID: id, Hash: hash})
	}
	return codes, nil
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

// ReplaceRecoveryCodeHash upgrades one stored hash in place.
func (r *MemoryRepository) ReplaceRecoveryCodeHash(
	ctx context.Context,
	id int64,
	hash string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, user := range r.users {
		for oldHash, codeID := range user.codes {
			if codeID == id {
				delete(user.codes, oldHash)
				user.codes[hash] = codeID
				return nil
			}
		}
	}
	return ErrNotFound
}

// LegacySecrets lists accounts whose secret is still stored in plaintext.
func (r *MemoryRepository) LegacySecrets(ctx context.Context, limit int) ([]LegacySecret, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	secrets := make([]LegacySecret, 0, len(r.users))
	for publicID, user := range r.users {
		if user.secret == "" || len(user.sealed) > 0 {
			continue
		}
		secrets = append(secrets, LegacySecret{PublicID: publicID, Secret: user.secret})
		if limit > 0 && len(secrets) >= limit {
			break
		}
	}
	return secrets, nil
}
