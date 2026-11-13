package audit

import "context"

type Recorder interface {
	Record(context.Context, Event) error
}

type Reader interface {
	List(context.Context, ListQuery) ([]Event, error)
}
