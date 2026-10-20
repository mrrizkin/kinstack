// Package log provides a structured logger built on zerolog that implements
// Fiber's AllLogger interface. It supports console and rolling-file output
// with configurable log levels and JSON formatting.
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

// Logger is a structured logger built on zerolog that satisfies Fiber's AllLogger interface.
type Logger struct {
	core zerolog.Logger
}

type LoggerDependency struct {
	Config config.Config
	// ExtraWriters receive the same JSON lines as the rolling file, for
	// example an OpenTelemetry log exporter. They never replace file logging.
	ExtraWriters []io.Writer
}

// NewLogger creates a Logger from the application config.
// It requires file logging to be enabled and panics otherwise.
func NewLogger(d LoggerDependency) *Logger {
	assert.Assert(d.Config.Log.File, "file logging must be enabled")
	fw := assert.OK(rollingFile(d.Config))
	var writer io.Writer = fw
	if len(d.ExtraWriters) > 0 {
		writer = io.MultiWriter(append([]io.Writer{fw}, d.ExtraWriters...)...)
	}

	// Configure zerolog
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

// Scope returns a new Logger with a "scope" field attached,
// useful for distinguishing log output from different subsystems.
func (l *Logger) Scope(name string) *Logger {
	return &Logger{
		core: l.core.With().Str("scope", name).Logger(),
	}
}

// GetLevel returns the current logging level as a human-readable string.
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

// GetLogger returns the underlying zerolog.Logger for direct use.
func (l *Logger) GetLogger() *zerolog.Logger {
	return &l.core
}

// -----------------------------------------------------------------------------
// Fiber adapter
// -----------------------------------------------------------------------------

// Ensure Logger implements Fiber's AllLogger interface and Log.
var (
	_ fiberlog.AllLogger[*Logger] = (*Logger)(nil)
	_ Log                         = (*Logger)(nil)
)

// Logger returns the receiver itself, satisfying Fiber's AllLogger interface.
func (l *Logger) Logger() *Logger {
	return l
}

// SetLevel changes the global zerolog logging level to match Fiber's level enum.
// WARNING: This affects ALL Logger instances in the process, not just the receiver.
// The name is dictated by Fiber's AllLogger interface — the global scope is a
// framework limitation, not a design choice.
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

// SetOutput replaces all logger outputs with the given writer.
func (l *Logger) SetOutput(w io.Writer) {
	l.core = l.core.Output(w)
}

// WithContext returns a new Logger that includes the given context
// for correlated logging (e.g., request IDs, trace IDs).
func (l *Logger) WithContext(value any) fiberlog.CommonLogger {
	ctx, ok := value.(context.Context)
	if !ok {
		ctx = context.Background()
	}
	return &Logger{
		core: l.core.With().Ctx(ctx).Logger(),
	}
}

// -----------------------------------------------------------------------------
// Structured and formatted logging
// -----------------------------------------------------------------------------

// log writes a structured event with key-value pairs.
// Keys and values must come in pairs; keys must be strings.
// If a value is an error, its Error() string is logged instead.
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

// Tracew logs at Trace level with structured key-value pairs.
func (l *Logger) Tracew(msg string, keysAndValues ...any) {
	l.log(l.core.Trace(), msg, keysAndValues...)
}

// Debugw logs at Debug level with structured key-value pairs.
func (l *Logger) Debugw(msg string, keysAndValues ...any) {
	l.log(l.core.Debug(), msg, keysAndValues...)
}

// Infow logs at Info level with structured key-value pairs.
func (l *Logger) Infow(msg string, keysAndValues ...any) {
	l.log(l.core.Info(), msg, keysAndValues...)
}

// Warnw logs at Warn level with structured key-value pairs.
func (l *Logger) Warnw(msg string, keysAndValues ...any) {
	l.log(l.core.Warn(), msg, keysAndValues...)
}

// Errorw logs at Error level with structured key-value pairs.
func (l *Logger) Errorw(msg string, keysAndValues ...any) {
	l.log(l.core.Error(), msg, keysAndValues...)
}

// Fatalw logs at Fatal level with structured key-value pairs and exits.
func (l *Logger) Fatalw(msg string, keysAndValues ...any) {
	l.log(l.core.Fatal(), msg, keysAndValues...)
}

// Panicw logs at Panic level with structured key-value pairs and panics.
func (l *Logger) Panicw(msg string, keysAndValues ...any) {
	l.log(l.core.Panic(), msg, keysAndValues...)
}

// Tracef logs at Trace level using printf-style formatting.
func (l *Logger) Tracef(format string, v ...any) {
	l.core.Trace().Msg(fmt.Sprintf(format, v...))
}

// Debugf logs at Debug level using printf-style formatting.
func (l *Logger) Debugf(format string, v ...any) {
	l.core.Debug().Msg(fmt.Sprintf(format, v...))
}

// Infof logs at Info level using printf-style formatting.
func (l *Logger) Infof(format string, v ...any) {
	l.core.Info().Msg(fmt.Sprintf(format, v...))
}

// Warnf logs at Warn level using printf-style formatting.
func (l *Logger) Warnf(format string, v ...any) {
	l.core.Warn().Msg(fmt.Sprintf(format, v...))
}

// Errorf logs at Error level using printf-style formatting.
func (l *Logger) Errorf(format string, v ...any) {
	l.core.Error().Msg(fmt.Sprintf(format, v...))
}

// Fatalf logs at Fatal level using printf-style formatting.
func (l *Logger) Fatalf(format string, v ...any) {
	l.core.Fatal().Msg(fmt.Sprintf(format, v...))
}

// Panicf logs at Panic level using printf-style formatting.
func (l *Logger) Panicf(format string, v ...any) {
	l.core.Panic().Msg(fmt.Sprintf(format, v...))
}

// -----------------------------------------------------------------------------
// Simple logging
// -----------------------------------------------------------------------------

// Trace logs at Trace level. If the first argument is a string it is used as the message.
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

// Debug logs at Debug level. If the first argument is a string it is used as the message.
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

// Info logs at Info level. If the first argument is a string it is used as the message.
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

// Warn logs at Warn level. If the first argument is a string it is used as the message.
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

// Error logs at Error level. If the first argument is a string it is used as the message.
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

// Fatal logs at Fatal level and calls os.Exit(1). If the first argument is a string it is used as
// the message.
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

// Panic logs at Panic level and panics. If the first argument is a string it is used as the
// message.
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
