package audit

import (
	"context"
	"sort"
	"sync"
)

// MemoryRepository stores audit records in memory for tests and local runs.
type MemoryRepository struct {
	mu     sync.RWMutex
	nextID int64
	events []AuditRecord
}

var _ Repository = (*MemoryRepository)(nil)

// NewMemoryRepository creates an empty repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{nextID: 1, events: make([]AuditRecord, 0)}
}

// Insert appends a record and assigns its identifier.
func (r *MemoryRepository) Insert(ctx context.Context, record AuditRecord) (AuditRecord, error) {
	if err := ctx.Err(); err != nil {
		return AuditRecord{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	record.ID = r.nextID
	r.nextID++
	record.Metadata = cloneMetadata(record.Metadata)
	r.events = append(r.events, record)
	return recordWithClonedMetadata(record), nil
}

// List returns matching records in reverse chronological order.
func (r *MemoryRepository) List(ctx context.Context, query RepositoryQuery) ([]AuditRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateLimit(query.Limit); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	records := make([]AuditRecord, 0, len(r.events))
	for _, record := range r.events {
		if query.ActorKey != "" && record.ActorKey != query.ActorKey {
			continue
		}
		if query.Action != "" && record.Action != query.Action {
			continue
		}
		records = append(records, recordWithClonedMetadata(record))
	}
	sort.SliceStable(records, func(i, j int) bool {
		if records[i].OccurredAt.Equal(records[j].OccurredAt) {
			return records[i].ID > records[j].ID
		}
		return records[i].OccurredAt.After(records[j].OccurredAt)
	})
	if query.Limit > 0 && len(records) > query.Limit {
		records = records[:query.Limit]
	}
	return records, nil
}

func recordWithClonedMetadata(record AuditRecord) AuditRecord {
	record.Metadata = cloneMetadata(record.Metadata)
	return record
}
