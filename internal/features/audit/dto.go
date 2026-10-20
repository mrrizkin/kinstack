package audit

// EntryDTO is the public contract for one audit event.
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

// Filters is the normalized filter set echoed back to the page.
type Filters struct {
	ActorKey string `json:"actor_key"`
	Action   string `json:"action"`
	Outcome  string `json:"outcome"`
	Search   string `json:"search"`
	From     string `json:"from"`
	To       string `json:"to"`
}

// PageProps is the public contract for the audit viewer page.
type PageProps struct {
	Entries []EntryDTO `json:"entries"`
	Filters Filters    `json:"filters"`
	Total   int        `json:"total"`
	Page    int        `json:"page"`
	PerPage int        `json:"per_page"`
	Pages   int        `json:"pages"`
}

// FilterRequest is the raw query input accepted by the service.
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
