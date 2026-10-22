package audit

import "kinstack/internal/pkg/pagination"

type EntryDTO struct {
	ID         int64             `json:"id"`
	OccurredAt string            `json:"occurred_at"`
	RequestID  string            `json:"request_id"`
	ActorKey   string            `json:"actor_key"`
	Action     string            `json:"action"`
	Resource   string            `json:"resource"`
	Method     string            `json:"method"`
	Status     int               `json:"status"`
	Outcome    string            `json:"outcome"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Filters is the filter set echoed back to the page. Date bounds are echoed as
// requested, so a date-only `to` stays inclusive and the viewer reopens on the
// day the user picked.
type Filters struct {
	ActorKey string `json:"actor_key"`
	Action   string `json:"action"`
	Outcome  string `json:"outcome"`
	Search   string `json:"search"`
	From     string `json:"from"`
	To       string `json:"to"`
}

type PageProps struct {
	Entries []EntryDTO      `json:"entries"`
	Filters Filters         `json:"filters"`
	Page    pagination.Page `json:"page"`
}

type FilterRequest struct {
	ActorKey string
	Action   string
	Outcome  string
	Search   string
	From     string
	To       string
	Page     int
	PerPage  int
}
