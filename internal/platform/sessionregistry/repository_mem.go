package sessionregistry

import (
	"context"
	"sort"
	"sync"
	"time"
)

// MemoryRepository is the deterministic in-memory session registry used by
// tests. It is never selected by production wiring.
type MemoryRepository struct {
	mu      sync.Mutex
	users   map[string]struct{}
	records map[string]Result
}

var _ Repository = (*MemoryRepository)(nil)

// NewMemoryRepository constructs an isolated-test registry with the supplied
// known user public IDs.
func NewMemoryRepository(userPublicIDs ...string) *MemoryRepository {
	repository := &MemoryRepository{
		users:   make(map[string]struct{}, len(userPublicIDs)),
		records: map[string]Result{},
	}
	for _, publicID := range userPublicIDs {
		repository.users[publicID] = struct{}{}
	}
	return repository
}

// Register records one session.
func (r *MemoryRepository) Register(
	_ context.Context,
	sessionPublicID string,
	input RegisterInput,
) (Result, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[input.UserPublicID]; !ok {
		return Result{}, ErrNotFound
	}
	for publicID, record := range r.records {
		if record.SessionID == input.SessionID {
			record.RevokedAt = nil
			record.IP = input.IP
			record.UserAgent = input.UserAgent
			record.LastSeenAt = time.Now().UTC()
			r.records[publicID] = record
			return record, nil
		}
	}
	now := time.Now().UTC()
	record := Result{
		PublicID:     sessionPublicID,
		UserPublicID: input.UserPublicID,
		SessionID:    input.SessionID,
		IP:           input.IP,
		UserAgent:    input.UserAgent,
		CreatedAt:    now,
		LastSeenAt:   now,
	}
	r.records[sessionPublicID] = record
	return record, nil
}

// Touch refreshes last_seen_at.
func (r *MemoryRepository) Touch(ctx context.Context, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for publicID, record := range r.records {
		if record.SessionID == sessionID && record.RevokedAt == nil {
			record.LastSeenAt = time.Now().UTC()
			r.records[publicID] = record
		}
	}
	return nil
}

// ListActive returns the active sessions of one user.
func (r *MemoryRepository) ListActive(ctx context.Context, userPublicID string) ([]Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	results := make([]Result, 0)
	for _, record := range r.records {
		if record.UserPublicID == userPublicID && record.RevokedAt == nil {
			results = append(results, record)
		}
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].LastSeenAt.After(results[j].LastSeenAt)
	})
	return results, nil
}

// FindByPublicID returns one active session.
func (r *MemoryRepository) FindByPublicID(ctx context.Context, sessionPublicID string) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[sessionPublicID]
	if !ok || record.RevokedAt != nil {
		return Result{}, ErrNotFound
	}
	return record, nil
}

// Revoke marks one session as revoked.
func (r *MemoryRepository) Revoke(ctx context.Context, sessionPublicID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.records[sessionPublicID]
	if !ok || record.RevokedAt != nil {
		return ErrNotFound
	}
	now := time.Now().UTC()
	record.RevokedAt = &now
	r.records[sessionPublicID] = record
	return nil
}

// RevokeBySessionID marks the session with the given transport ID as revoked.
func (r *MemoryRepository) RevokeBySessionID(ctx context.Context, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	for publicID, record := range r.records {
		if record.SessionID == sessionID && record.RevokedAt == nil {
			record.RevokedAt = &now
			r.records[publicID] = record
		}
	}
	return nil
}

// RevokeAllExcept revokes every session except the kept transport ID.
func (r *MemoryRepository) RevokeAllExcept(
	ctx context.Context,
	userPublicID string,
	keepSessionID string,
) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	sessionIDs := make([]string, 0)
	for publicID, record := range r.records {
		if record.UserPublicID != userPublicID || record.RevokedAt != nil {
			continue
		}
		if record.SessionID == keepSessionID {
			continue
		}
		record.RevokedAt = &now
		r.records[publicID] = record
		sessionIDs = append(sessionIDs, record.SessionID)
	}
	return sessionIDs, nil
}
