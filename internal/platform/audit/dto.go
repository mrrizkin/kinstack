package audit

import "time"

// Action identifies an audited operation.
type Action string

// Outcome identifies the result of an audited operation.
type Outcome string

const (
	ActionHTTPRead      Action  = "http.read"
	ActionHTTPMutate    Action  = "http.mutate"
	ActionBackgroundJob Action  = "background.job"
	OutcomeSuccess      Outcome = "success"
	OutcomeFailure      Outcome = "failure"
)

// Event is the application-level audit event contract.
type Event struct {
	ID         int64             `json:"id"`
	OccurredAt time.Time         `json:"occurred_at"`
	RequestID  string            `json:"request_id"`
	ActorKey   string            `json:"actor_key"`
	Action     Action            `json:"action"`
	Resource   string            `json:"resource"`
	Method     string            `json:"method"`
	Status     int               `json:"status"`
	Outcome    Outcome           `json:"outcome"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ListQuery contains audit projection filters.
type ListQuery struct {
	ActorKey string
	Action   Action
	Limit    int
}
