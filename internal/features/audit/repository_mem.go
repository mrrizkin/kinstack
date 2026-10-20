package audit

import (
	"context"
	"sort"
	"strings"
	"sync"
)

// MemoryRepository is the deterministic in-memory repository used by service
// tests. It is never selected by production wiring.
type MemoryRepository struct {
	mu      sync.RWMutex
	nextID  int64
	entries []Entry
}

var _ Repository = (*MemoryRepository)(nil)

// NewMemoryRepository constructs an isolated-test audit viewer store.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{nextID: 1, entries: make([]Entry, 0)}
}

// Seed inserts one entry into the memory store.
func (r *MemoryRepository) Seed(entry Entry) Entry {
	r.mu.Lock()
	defer r.mu.Unlock()

	if entry.ID == 0 {
		entry.ID = r.nextID
		r.nextID++
	}
	r.entries = append(r.entries, cloneEntry(entry))
	return cloneEntry(entry)
}

// List returns matching entries in reverse chronological order.
func (r *MemoryRepository) List(_ context.Context, query Query) (Page, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	matches := make([]Entry, 0, len(r.entries))
	for _, entry := range r.entries {
		if !matchesQuery(entry, query) {
			continue
		}
		matches = append(matches, cloneEntry(entry))
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].OccurredAt.Equal(matches[j].OccurredAt) {
			return matches[i].ID > matches[j].ID
		}
		return matches[i].OccurredAt.After(matches[j].OccurredAt)
	})

	total := len(matches)
	if query.Offset >= total {
		return Page{Entries: []Entry{}, Total: total}, nil
	}
	end := query.Offset + query.Limit
	if query.Limit <= 0 || end > total {
		end = total
	}
	return Page{Entries: matches[query.Offset:end], Total: total}, nil
}

func matchesQuery(entry Entry, query Query) bool {
	if query.ActorKey != "" && entry.ActorKey != query.ActorKey {
		return false
	}
	if query.Action != "" && entry.Action != query.Action {
		return false
	}
	if query.Outcome != "" && entry.Outcome != query.Outcome {
		return false
	}
	if query.From != nil && entry.OccurredAt.Before(*query.From) {
		return false
	}
	if query.To != nil && !entry.OccurredAt.Before(*query.To) {
		return false
	}
	if query.Search != "" {
		needle := strings.ToLower(query.Search)
		haystack := strings.ToLower(strings.Join(
			[]string{entry.Resource, entry.RequestID, entry.ActorKey, entry.Action},
			" ",
		))
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}

func cloneEntry(entry Entry) Entry {
	if len(entry.Metadata) == 0 {
		entry.Metadata = map[string]string{}
		return entry
	}
	metadata := make(map[string]string, len(entry.Metadata))
	for key, value := range entry.Metadata {
		metadata[key] = value
	}
	entry.Metadata = metadata
	return entry
}
