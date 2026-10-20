package audit

import "context"

// Recorder appends audit events.
type Recorder interface {
	Record(context.Context, Event) error
}

// Reader reads audit events independently from the write port.
type Reader interface {
	List(context.Context, ListQuery) ([]Event, error)
}
