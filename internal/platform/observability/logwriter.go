package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
)

const maxLogLineBytes = 1 << 20

// LogWriter is an io.Writer that turns zerolog JSON lines into OpenTelemetry
// log records. It buffers partial lines because zerolog writers may split a
// record across Write calls.
type LogWriter struct {
	mu     sync.Mutex
	buffer []byte
	logger otellog.Logger
}

// NewLogWriter creates a writer that emits records to logger.
func NewLogWriter(logger otellog.Logger) *LogWriter {
	return &LogWriter{logger: logger}
}

// Write implements io.Writer.
func (w *LogWriter) Write(p []byte) (int, error) {
	if w == nil || w.logger == nil {
		return len(p), nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buffer = append(w.buffer, p...)
	for {
		index := bytes.IndexByte(w.buffer, '\n')
		if index < 0 {
			break
		}
		line := make([]byte, index)
		copy(line, w.buffer[:index])
		w.buffer = w.buffer[index+1:]
		w.emit(line)
	}
	if len(w.buffer) > maxLogLineBytes {
		line := w.buffer
		w.buffer = nil
		w.emit(line)
	}
	return len(p), nil
}

// Close emits any buffered partial line. It does not shut down the logger
// provider; Telemetry.Shutdown owns that.
func (w *LogWriter) Close() {
	if w == nil {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.buffer) > 0 {
		line := w.buffer
		w.buffer = nil
		w.emit(line)
	}
}

func (w *LogWriter) emit(line []byte) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return
	}

	fields := map[string]any{}
	if err := json.Unmarshal(line, &fields); err != nil {
		w.emitRecord(time.Now(), otellog.SeverityInfo, "info", string(line), nil)
		return
	}

	timestamp := recordTime(fields["time"])
	level := strings.ToLower(strings.TrimSpace(stringField(fields["level"])))
	message := stringField(fields["message"])
	if message == "" {
		message = string(line)
	}

	attributes := make([]attribute.KeyValue, 0, len(fields))
	for key, value := range fields {
		switch key {
		case "time", "level", "message":
			continue
		}
		attributes = append(attributes, attribute.String(key, stringify(value)))
	}
	w.emitRecord(timestamp, severity(level), level, message, attributes)
}

func (w *LogWriter) emitRecord(
	timestamp time.Time,
	level otellog.Severity,
	levelText string,
	body string,
	attributes []attribute.KeyValue,
) {
	var record otellog.Record
	record.SetTimestamp(timestamp)
	record.SetObservedTimestamp(time.Now())
	record.SetSeverity(level)
	record.SetSeverityText(strings.ToUpper(levelText))
	record.SetBody(attribute.StringValue(body))
	record.AddAttributes(attributes...)
	w.logger.Emit(context.Background(), record)
}

func severity(level string) otellog.Severity {
	switch level {
	case "trace":
		return otellog.SeverityTrace
	case "debug":
		return otellog.SeverityDebug
	case "warn", "warning":
		return otellog.SeverityWarn
	case "error":
		return otellog.SeverityError
	case "fatal", "panic":
		return otellog.SeverityFatal
	default:
		return otellog.SeverityInfo
	}
}

func recordTime(value any) time.Time {
	switch typed := value.(type) {
	case float64:
		return unixTime(int64(typed))
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return unixTime(parsed)
		}
	case string:
		if parsed, err := time.Parse(time.RFC3339Nano, typed); err == nil {
			return parsed
		}
		if parsed, err := strconv.ParseInt(typed, 10, 64); err == nil {
			return unixTime(parsed)
		}
	}
	return time.Now()
}

func unixTime(value int64) time.Time {
	if value > 1e12 {
		return time.UnixMilli(value)
	}
	return time.Unix(value, 0)
}

func stringField(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func stringify(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}
