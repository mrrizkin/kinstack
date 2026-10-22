// Package audit provides append-only audit event handling.
package audit

import (
	"context"
	"strings"
	"time"

	"kinstack/internal/pkg/assert"
)

// Audit validates events and delegates persistence to a Repository.
type Audit struct {
	repository Repository
}

// New creates an Audit service backed by repository.
func New(repository Repository) *Audit {
	assert.Assert(repository != nil, "audit: repository is required")

	return &Audit{repository: repository}
}

var (
	_ Recorder = (*Audit)(nil)
	_ Reader   = (*Audit)(nil)
)

// Record validates and appends an event.
func (a *Audit) Record(ctx context.Context, event Event) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	event = normalizeEvent(event)
	if err := validateEvent(event); err != nil {
		return err
	}
	_, err := a.repository.Insert(ctx, AuditRecord{
		OccurredAt: event.OccurredAt,
		RequestID:  event.RequestID,
		ActorKey:   event.ActorKey,
		Action:     event.Action,
		Resource:   event.Resource,
		Method:     event.Method,
		Status:     event.Status,
		Outcome:    event.Outcome,
		Metadata:   cloneMetadata(event.Metadata),
	})
	return err
}

// Prune deletes audit records older than the cutoff. It is meant for a
// scheduled maintenance job; the request path never calls it.
func (a *Audit) Prune(ctx context.Context, before time.Time) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if before.IsZero() {
		return 0, ErrInvalidEvent
	}
	return a.repository.DeleteBefore(ctx, before)
}

func (a *Audit) List(ctx context.Context, query ListQuery) ([]Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateLimit(query.Limit); err != nil {
		return nil, err
	}
	records, err := a.repository.List(ctx, RepositoryQuery(query))
	if err != nil {
		return nil, err
	}
	events := make([]Event, 0, len(records))
	for _, record := range records {
		events = append(events, eventFromRecord(record))
	}
	return events, nil
}

// -----------------------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------------------

func validateEvent(event Event) error {
	if strings.TrimSpace(string(event.Action)) == "" || strings.TrimSpace(event.Resource) == "" ||
		strings.TrimSpace(event.Method) == "" || event.Status < 100 || event.Status > 599 {
		return ErrInvalidEvent
	}
	if event.Outcome != OutcomeSuccess && event.Outcome != OutcomeFailure {
		return ErrInvalidEvent
	}
	if len(event.Action) > 100 || len(event.Resource) > 500 || len(event.Method) > 16 ||
		len(event.RequestID) > 200 || len(event.ActorKey) > 200 {
		return ErrInvalidEvent
	}
	for key, value := range event.Metadata {
		if len(key) == 0 || len(key) > 100 || len(value) > 500 {
			return ErrInvalidEvent
		}
	}
	return nil
}

func normalizeEvent(event Event) Event {
	event.Action = Action(strings.TrimSpace(string(event.Action)))
	event.Resource = strings.TrimSpace(event.Resource)
	event.Method = strings.ToUpper(strings.TrimSpace(event.Method))
	event.RequestID = strings.TrimSpace(event.RequestID)
	event.ActorKey = strings.TrimSpace(event.ActorKey)
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	} else {
		event.OccurredAt = event.OccurredAt.UTC()
	}
	if event.Metadata == nil {
		event.Metadata = map[string]string{}
	}
	return event
}

func validateLimit(limit int) error {
	if limit < 0 || limit > 1000 {
		return ErrInvalidEvent
	}
	return nil
}

func eventFromRecord(record AuditRecord) Event {
	return Event{
		ID:         record.ID,
		OccurredAt: record.OccurredAt,
		RequestID:  record.RequestID,
		ActorKey:   record.ActorKey,
		Action:     record.Action,
		Resource:   record.Resource,
		Method:     record.Method,
		Status:     record.Status,
		Outcome:    record.Outcome,
		Metadata:   cloneMetadata(record.Metadata),
	}
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return map[string]string{}
	}
	copyOf := make(map[string]string, len(metadata))
	for key, value := range metadata {
		copyOf[key] = value
	}
	return copyOf
}
