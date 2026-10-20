package audit

import (
	"context"
	"time"
)

// Repository persists append-only audit records.
type Repository interface {
	Insert(context.Context, AuditRecord) (AuditRecord, error)
	List(context.Context, RepositoryQuery) ([]AuditRecord, error)
	// DeleteBefore permanently removes records older than the cutoff and
	// reports how many rows were deleted.
	DeleteBefore(context.Context, time.Time) (int64, error)
}

// AuditRecord is the repository representation of an audit event.
type AuditRecord struct {
	ID         int64
	OccurredAt time.Time
	RequestID  string
	ActorKey   string
	Action     Action
	Resource   string
	Method     string
	Status     int
	Outcome    Outcome
	Metadata   map[string]string
}

// RepositoryQuery contains repository-level filters.
type RepositoryQuery struct {
	ActorKey string
	Action   Action
	Limit    int
}
