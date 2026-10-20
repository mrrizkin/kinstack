package audit

import "context"

type NoopRecorder struct{}

func (NoopRecorder) Record(context.Context, Event) error { return nil }
