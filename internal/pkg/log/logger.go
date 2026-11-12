// Package log is the structured logger: zerolog behind Fiber's AllLogger
// interface, with rolling-file output.
package log

import (
	"context"
	"fmt"
	"io"

	"kinstack/config"
	"kinstack/internal/pkg/assert"

	fiberlog "github.com/gofiber/fiber/v3/log"
	"github.com/rs/zerolog"
)

type Logger struct {
	core zerolog.Logger
}

type LoggerDependency struct {
	Config config.Config
	// ExtraWriters receive the same JSON lines as the rolling file, for
	// example an OpenTelemetry log exporter. They never replace file logging.
	ExtraWriters []io.Writer
}

// NewWriterLogger writes to w; tests use it to assert log output.
func NewWriterLogger(w io.Writer) *Logger {
	assert.Assert(w != nil, "log writer is required")
	return &Logger{core: zerolog.New(w).With().Timestamp().Logger()}
}

// NewLogger requires file logging and panics otherwise.
func NewLogger(d LoggerDependency) *Logger {
	assert.Assert(d.Config.Log.File, "file logging must be enabled")
	fw := assert.OK(rollingFile(d.Config))
	var writer io.Writer = fw
	if len(d.ExtraWriters) > 0 {
		writer = io.MultiWriter(append([]io.Writer{fw}, d.ExtraWriters...)...)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	switch d.Config.Log.Level {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	case "none":
		zerolog.SetGlobalLevel(zerolog.NoLevel)
	case "disable":
		zerolog.SetGlobalLevel(zerolog.Disabled)
	default:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	logger := zerolog.New(writer).
		With().
		Timestamp().
		Logger()

	return &Logger{
		core: logger,
	}
}

// Scope tags every line with a subsystem name.
func (l *Logger) Scope(name string) *Logger {
	return &Logger{
		core: l.core.With().Str("scope", name).Logger(),
	}
}

func (l *Logger) GetLevel() string {
	switch l.core.GetLevel() {
	case zerolog.TraceLevel:
		return "trace"
	case zerolog.DebugLevel:
		return "debug"
	case zerolog.InfoLevel:
		return "info"
	case zerolog.WarnLevel:
		return "warn"
	case zerolog.ErrorLevel:
		return "error"
	case zerolog.FatalLevel:
		return "fatal"
	case zerolog.PanicLevel:
		return "panic"
	case zerolog.NoLevel:
		return "none"
	case zerolog.Disabled:
		return "disable"
	default:
		return "none"
	}
}

func (l *Logger) GetLogger() *zerolog.Logger {
	return &l.core
}

var (
	_ fiberlog.AllLogger[*Logger] = (*Logger)(nil)
	_ Log                         = (*Logger)(nil)
)

func (l *Logger) Logger() *Logger {
	return l
}

// SetLevel changes the global zerolog level for the whole process: the name
// comes from Fiber's interface, not from this package.
func (l *Logger) SetLevel(level fiberlog.Level) {
	var zLevel zerolog.Level
	switch level {
	case fiberlog.LevelTrace:
		zLevel = zerolog.TraceLevel
	case fiberlog.LevelDebug:
		zLevel = zerolog.DebugLevel
	case fiberlog.LevelInfo:
		zLevel = zerolog.InfoLevel
	case fiberlog.LevelWarn:
		zLevel = zerolog.WarnLevel
	case fiberlog.LevelError:
		zLevel = zerolog.ErrorLevel
	case fiberlog.LevelFatal:
		zLevel = zerolog.FatalLevel
	case fiberlog.LevelPanic:
		zLevel = zerolog.PanicLevel
	default:
		zLevel = zerolog.DebugLevel
	}
	zerolog.SetGlobalLevel(zLevel)
}

func (l *Logger) SetOutput(w io.Writer) {
	l.core = l.core.Output(w)
}

// WithContext attaches the context for correlated logging.
func (l *Logger) WithContext(value any) fiberlog.CommonLogger {
	ctx, ok := value.(context.Context)
	if !ok {
		ctx = context.Background()
	}
	return &Logger{
		core: l.core.With().Ctx(ctx).Logger(),
	}
}

// RequestIDKey is the field every request-scoped log line carries.
const RequestIDKey = "request_id"

// ForRequest binds the request ID so no call site passes it by hand.
func ForRequest(logger Log, requestID string) Log {
	concrete, ok := logger.(*Logger)
	if !ok || requestID == "" {
		return logger
	}
	return &Logger{core: concrete.core.With().Str(RequestIDKey, requestID).Logger()}
}

// log asserts pairs of string keys and values; errors are logged as strings.
func (l *Logger) log(event *zerolog.Event, msg string, keysAndValues ...any) {
	for i := 0; i < len(keysAndValues); i += 2 {
		assert.Assert(i+1 < len(keysAndValues), "keysAndValues must be in pairs")
		key, ok := keysAndValues[i].(string)
		assert.Assert(ok, "keys should be strings")

		if val, ok := keysAndValues[i+1].(error); ok {
			event.Interface(key, val.Error())
		} else {
			event.Interface(key, keysAndValues[i+1])
		}
	}
	event.Msg(msg)
}

func (l *Logger) Tracew(msg string, keysAndValues ...any) {
	l.log(l.core.Trace(), msg, keysAndValues...)
}

func (l *Logger) Debugw(msg string, keysAndValues ...any) {
	l.log(l.core.Debug(), msg, keysAndValues...)
}

func (l *Logger) Infow(msg string, keysAndValues ...any) {
	l.log(l.core.Info(), msg, keysAndValues...)
}

func (l *Logger) Warnw(msg string, keysAndValues ...any) {
	l.log(l.core.Warn(), msg, keysAndValues...)
}

func (l *Logger) Errorw(msg string, keysAndValues ...any) {
	l.log(l.core.Error(), msg, keysAndValues...)
}

// Fatalw logs and exits.
func (l *Logger) Fatalw(msg string, keysAndValues ...any) {
	l.log(l.core.Fatal(), msg, keysAndValues...)
}

// Panicw logs and panics.
func (l *Logger) Panicw(msg string, keysAndValues ...any) {
	l.log(l.core.Panic(), msg, keysAndValues...)
}

func (l *Logger) Tracef(format string, v ...any) {
	l.core.Trace().Msg(fmt.Sprintf(format, v...))
}

func (l *Logger) Debugf(format string, v ...any) {
	l.core.Debug().Msg(fmt.Sprintf(format, v...))
}

func (l *Logger) Infof(format string, v ...any) {
	l.core.Info().Msg(fmt.Sprintf(format, v...))
}

func (l *Logger) Warnf(format string, v ...any) {
	l.core.Warn().Msg(fmt.Sprintf(format, v...))
}

// Errorf logs at Error level using printf-style formatting.
func (l *Logger) Errorf(format string, v ...any) {
	l.core.Error().Msg(fmt.Sprintf(format, v...))
}

func (l *Logger) Fatalf(format string, v ...any) {
	l.core.Fatal().Msg(fmt.Sprintf(format, v...))
}

func (l *Logger) Panicf(format string, v ...any) {
	l.core.Panic().Msg(fmt.Sprintf(format, v...))
}

// A leading string argument is the message; the rest are key-value pairs.
func (l *Logger) Trace(v ...any) {
	if len(v) == 0 {
		return
	}
	if msg, ok := v[0].(string); ok {
		l.log(l.core.Trace(), msg, v[1:]...)
	} else {
		l.core.Trace().Msg(fmt.Sprint(v...))
	}
}

func (l *Logger) Debug(v ...any) {
	if len(v) == 0 {
		return
	}
	if msg, ok := v[0].(string); ok {
		l.log(l.core.Debug(), msg, v[1:]...)
	} else {
		l.core.Debug().Msg(fmt.Sprint(v...))
	}
}

func (l *Logger) Info(v ...any) {
	if len(v) == 0 {
		return
	}
	if msg, ok := v[0].(string); ok {
		l.log(l.core.Info(), msg, v[1:]...)
	} else {
		l.core.Info().Msg(fmt.Sprint(v...))
	}
}

func (l *Logger) Warn(v ...any) {
	if len(v) == 0 {
		return
	}
	if msg, ok := v[0].(string); ok {
		l.log(l.core.Warn(), msg, v[1:]...)
	} else {
		l.core.Warn().Msg(fmt.Sprint(v...))
	}
}

func (l *Logger) Error(v ...any) {
	if len(v) == 0 {
		return
	}
	if msg, ok := v[0].(string); ok {
		l.log(l.core.Error(), msg, v[1:]...)
	} else {
		l.core.Error().Msg(fmt.Sprint(v...))
	}
}

// Fatal logs and calls os.Exit(1).
func (l *Logger) Fatal(v ...any) {
	if len(v) == 0 {
		return
	}
	if msg, ok := v[0].(string); ok {
		l.log(l.core.Fatal(), msg, v[1:]...)
	} else {
		l.core.Fatal().Msg(fmt.Sprint(v...))
	}
}

// Panic logs and panics.
func (l *Logger) Panic(v ...any) {
	if len(v) == 0 {
		return
	}
	if msg, ok := v[0].(string); ok {
		l.log(l.core.Panic(), msg, v[1:]...)
	} else {
		l.core.Panic().Msg(fmt.Sprint(v...))
	}
}
