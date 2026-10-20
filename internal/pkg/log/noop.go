package log

// NoopLogger discards log entries. It is useful for isolated tests that still
// need to satisfy mandatory logger dependencies.
type NoopLogger struct{}

var _ Log = NoopLogger{}

func (NoopLogger) Tracew(string, ...any) {}
func (NoopLogger) Debugw(string, ...any) {}
func (NoopLogger) Infow(string, ...any)  {}
func (NoopLogger) Warnw(string, ...any)  {}
func (NoopLogger) Errorw(string, ...any) {}
func (NoopLogger) Fatalw(string, ...any) {}
func (NoopLogger) Panicw(string, ...any) {}

func (NoopLogger) Tracef(string, ...any) {}
func (NoopLogger) Debugf(string, ...any) {}
func (NoopLogger) Infof(string, ...any)  {}
func (NoopLogger) Warnf(string, ...any)  {}
func (NoopLogger) Errorf(string, ...any) {}
func (NoopLogger) Fatalf(string, ...any) {}
func (NoopLogger) Panicf(string, ...any) {}

func (NoopLogger) Trace(...any) {}
func (NoopLogger) Debug(...any) {}
func (NoopLogger) Info(...any)  {}
func (NoopLogger) Warn(...any)  {}
func (NoopLogger) Error(...any) {}
func (NoopLogger) Fatal(...any) {}
func (NoopLogger) Panic(...any) {}

func NewNoopLogger() Log { return NoopLogger{} }
