package audit

import "time"

type Action string

type Outcome string

const (
	ActionHTTPRead      Action  = "http.read"
	ActionHTTPMutate    Action  = "http.mutate"
	ActionBackgroundJob Action  = "background.job"
	OutcomeSuccess      Outcome = "success"
	OutcomeFailure      Outcome = "failure"
)

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

type ListQuery struct {
	ActorKey string
	Action   Action
	Limit    int
}
