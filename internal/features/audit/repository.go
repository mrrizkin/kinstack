// Package audit owns the read-only audit log viewer over the append-only
// audit_events projection written by platform/audit.
package audit

import (
	"context"
	"time"

	"kinstack/internal/pkg/pagination"
)

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

type Query struct {
	pagination.Query
	ActorKey string
	Action   string
	Outcome  string
	Search   string
	From     *time.Time
	To       *time.Time
}

type Page struct {
	Entries []Entry
	Total   int
}

// Repository reads the append-only audit projection.
type Repository interface {
	List(context.Context, Query) (Page, error)
}
