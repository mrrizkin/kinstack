// Package audit owns the read-only audit log viewer over the append-only
// audit_events projection written by platform/audit.
package audit

import (
	"context"
	"time"
)

// Entry is one persisted audit event projection.
type Entry struct {
	ID         int64
	OccurredAt time.Time
	RequestID  string
	ActorKey   string
	Action     string
	Resource   string
	Method     string
	Status     int
	Outcome    string
	Metadata   map[string]string
}

// Query filters and paginates the audit projection.
type Query struct {
	ActorKey string
	Action   string
	Outcome  string
	Search   string
	From     *time.Time
	To       *time.Time
	Limit    int
	Offset   int
}

// Page is one page of audit entries plus the total number of matches.
type Page struct {
	Entries []Entry
	Total   int
}

// Repository reads the append-only audit projection.
type Repository interface {
	List(context.Context, Query) (Page, error)
}
