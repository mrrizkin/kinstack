package passkeys

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
)

const maxPasskeyNameLength = 80

// MemoryRepository is the deterministic in-memory repository used by service
// tests. It is not a production fallback.
type MemoryRepository struct {
	mu       sync.Mutex
	users    []FindPasskeyUserResult
	passkeys map[string][]PasskeyRecord
}

var _ Repository = (*MemoryRepository)(nil)

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		users: []FindPasskeyUserResult{
			{
				PublicID:    "A1b2C3d4E5f6G7h8I9j0K",
				Email:       "reza@example.com",
				Username:    "reza-pratama",
				DisplayName: "Reza Pratama",
				IsActive:    true,
			},
			{
				PublicID:    "C1d2E3f4G5h6I7j8K9l0M",
				Email:       "sari@example.com",
				Username:    "sari-wulandari",
				DisplayName: "Sari Wulandari",
				IsActive:    true,
			},
		},
		passkeys: map[string][]PasskeyRecord{},
	}
}

func (r *MemoryRepository) FindByPublicID(
	_ context.Context,
	id string,
) (FindPasskeyUserResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, user := range r.users {
		if user.PublicID == strings.TrimSpace(id) && activeUser(user) {
			return user, nil
		}
	}
	return FindPasskeyUserResult{}, ErrUserNotFound
}

func (r *MemoryRepository) ListPasskeys(
	_ context.Context,
	id string,
) ([]ListPasskeysResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.activeUserLocked(id) {
		return nil, ErrUserNotFound
	}
	records := r.passkeys[strings.TrimSpace(id)]
	results := make([]ListPasskeysResult, 0, len(records))
	for _, record := range records {
		result, err := passkeyResult(record)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func (r *MemoryRepository) SavePasskey(_ context.Context, id string, record PasskeyRecord) error {
	if len(record.ID) == 0 || !bytes.Equal(record.ID, record.Credential.ID) ||
		strings.TrimSpace(
			record.Name,
		) == "" || len(strings.TrimSpace(record.Name)) > maxPasskeyNameLength {
		return ErrPasskeyInvalid
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	id = strings.TrimSpace(id)
	if !r.activeUserLocked(id) {
		return ErrUserNotFound
	}
	for _, records := range r.passkeys {
		for _, existing := range records {
			if bytes.Equal(existing.ID, record.ID) {
				return ErrPasskeyConflict
			}
		}
	}
	record.Name = strings.TrimSpace(record.Name)
	r.passkeys[id] = append(r.passkeys[id], clonePasskeyRecord(record))
	return nil
}

func (r *MemoryRepository) UpdatePasskey(_ context.Context, id string, record PasskeyRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	id = strings.TrimSpace(id)
	if !r.activeUserLocked(id) {
		return ErrUserNotFound
	}
	for i, existing := range r.passkeys[id] {
		if bytes.Equal(existing.ID, record.ID) {
			if record.Name == "" {
				record.Name = existing.Name
			}
			if record.CreatedAt.IsZero() {
				record.CreatedAt = existing.CreatedAt
			}
			r.passkeys[id][i] = clonePasskeyRecord(record)
			return nil
		}
	}
	return ErrPasskeyNotFound
}

func (r *MemoryRepository) DeletePasskey(_ context.Context, id string, credentialID []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	id = strings.TrimSpace(id)
	if !r.activeUserLocked(id) {
		return ErrUserNotFound
	}
	for i, record := range r.passkeys[id] {
		if bytes.Equal(record.ID, credentialID) {
			r.passkeys[id] = append(r.passkeys[id][:i], r.passkeys[id][i+1:]...)
			return nil
		}
	}
	return ErrPasskeyNotFound
}

func (r *MemoryRepository) FindPasskeyUser(
	_ context.Context,
	credentialID, userHandle []byte,
) (FindPasskeyUserResult, []ListPasskeysResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, records := range r.passkeys {
		if !r.activeUserLocked(id) || !bytes.Equal(passkeyUserHandle(id), userHandle) {
			continue
		}
		for _, record := range records {
			if !bytes.Equal(record.ID, credentialID) {
				continue
			}
			var user FindPasskeyUserResult
			for _, candidate := range r.users {
				if candidate.PublicID == id {
					user = candidate
					break
				}
			}
			results := make([]ListPasskeysResult, 0, len(records))
			for _, item := range records {
				result, err := passkeyResult(item)
				if err != nil {
					return FindPasskeyUserResult{}, nil, err
				}
				results = append(results, result)
			}
			return user, results, nil
		}
	}
	return FindPasskeyUserResult{}, nil, ErrPasskeyNotFound
}

func passkeyResult(record PasskeyRecord) (ListPasskeysResult, error) {
	credential, err := json.Marshal(record.Credential)
	if err != nil {
		return ListPasskeysResult{}, err
	}
	return ListPasskeysResult{
		ID:         append([]byte(nil), record.ID...),
		Name:       record.Name,
		Credential: credential,
		CreatedAt:  record.CreatedAt,
		LastUsedAt: record.LastUsedAt,
	}, nil
}

func clonePasskeyRecord(record PasskeyRecord) PasskeyRecord {
	record.ID = append([]byte(nil), record.ID...)
	record.Credential.ID = append([]byte(nil), record.Credential.ID...)
	record.Credential.PublicKey = append([]byte(nil), record.Credential.PublicKey...)
	return record
}

func activeUser(user FindPasskeyUserResult) bool {
	return user.DeletedAt == nil && user.IsActive
}

func (r *MemoryRepository) activeUserLocked(id string) bool {
	id = strings.TrimSpace(id)
	for _, user := range r.users {
		if user.PublicID == id && activeUser(user) {
			return true
		}
	}
	return false
}
