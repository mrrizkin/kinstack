package observability

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
)

type captureLogger struct {
	embedded.Logger
	records []otellog.Record
}

func (c *captureLogger) Emit(_ context.Context, record otellog.Record) {
	c.records = append(c.records, record)
}

func (c *captureLogger) Enabled(context.Context, otellog.EnabledParameters) bool {
	return true
}

func TestLogWriterEmitsZerologLine(t *testing.T) {
	sink := &captureLogger{}
	writer := NewLogWriter(sink)
	line := `{"level":"warn","time":1700000000,"message":"login failed","request_id":"req-1","attempt":3}` + "\n"
	if _, err := writer.Write([]byte(line)); err != nil {
		t.Fatalf("write: %v", err)
	}

	if len(sink.records) != 1 {
		t.Fatalf("records=%d, want 1", len(sink.records))
	}
	record := sink.records[0]
	if record.Severity() != otellog.SeverityWarn {
		t.Fatalf("severity=%v, want warn", record.Severity())
	}
	if record.SeverityText() != "WARN" {
		t.Fatalf("severity text=%q, want WARN", record.SeverityText())
	}
	if record.Body().AsString() != "login failed" {
		t.Fatalf("body=%q, want login failed", record.Body().AsString())
	}
	if record.Timestamp().Unix() != 1700000000 {
		t.Fatalf("timestamp=%v, want unix 1700000000", record.Timestamp())
	}

	attributes := map[string]string{}
	record.WalkAttributes(func(attr attribute.KeyValue) bool {
		attributes[string(attr.Key)] = attr.Value.AsString()
		return true
	})
	if attributes["request_id"] != "req-1" || attributes["attempt"] != "3" {
		t.Fatalf("unexpected attributes: %#v", attributes)
	}
}

func TestLogWriterBuffersPartialLines(t *testing.T) {
	sink := &captureLogger{}
	writer := NewLogWriter(sink)
	if _, err := writer.Write([]byte(`{"level":"info","message":"part`)); err != nil {
		t.Fatalf("write first: %v", err)
	}
	if len(sink.records) != 0 {
		t.Fatalf("records=%d before newline, want 0", len(sink.records))
	}
	if _, err := writer.Write([]byte(`ial"}` + "\n")); err != nil {
		t.Fatalf("write second: %v", err)
	}
	if len(sink.records) != 1 {
		t.Fatalf("records=%d, want 1", len(sink.records))
	}
	if sink.records[0].Body().AsString() != "partial" {
		t.Fatalf("body=%q, want partial", sink.records[0].Body().AsString())
	}
}

func TestLogWriterFallsBackToRawLine(t *testing.T) {
	sink := &captureLogger{}
	writer := NewLogWriter(sink)
	if _, err := writer.Write([]byte("not json\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(sink.records) != 1 {
		t.Fatalf("records=%d, want 1", len(sink.records))
	}
	if sink.records[0].Body().AsString() != "not json" {
		t.Fatalf("body=%q, want raw line", sink.records[0].Body().AsString())
	}
}

func TestLogWriterCloseFlushesBuffer(t *testing.T) {
	sink := &captureLogger{}
	writer := NewLogWriter(sink)
	if _, err := writer.Write([]byte(`{"level":"error","message":"tail"}`)); err != nil {
		t.Fatalf("write: %v", err)
	}
	writer.Close()
	if len(sink.records) != 1 || sink.records[0].Severity() != otellog.SeverityError {
		t.Fatalf("unexpected records: %#v", sink.records)
	}
}

func TestTelemetryNilIsSafe(t *testing.T) {
	var telemetry *Telemetry
	if telemetry.Enabled() {
		t.Fatal("nil telemetry reported enabled")
	}
	if telemetry.LogWriter() != nil {
		t.Fatal("nil telemetry returned a log writer")
	}
	if err := telemetry.Shutdown(context.Background()); err != nil {
		t.Fatalf("nil telemetry shutdown: %v", err)
	}
	handler := telemetry.Middleware()
	if handler == nil {
		t.Fatal("nil telemetry returned a nil middleware")
	}
}
