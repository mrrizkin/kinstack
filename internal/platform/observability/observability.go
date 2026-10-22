// Package observability wires opt-in error tracking (GlitchTip/Sentry) and
// OpenTelemetry traces, logs, and metrics for OpenObserve.
//
// Every integration is disabled unless its configuration is explicitly
// enabled, so the base keeps working with no external observability backend.
package observability

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"kinstack/config"

	"github.com/getsentry/sentry-go"
	"github.com/gofiber/fiber/v3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const traceIDLocal = "otel_trace_id"

type Dependency struct {
	Config config.Config
}

// Telemetry owns the optional observability clients. A zero-value Telemetry is
// a safe no-op.
type Telemetry struct {
	tracer         trace.Tracer
	tracerProvider *sdktrace.TracerProvider
	loggerProvider *sdklog.LoggerProvider
	logWriter      *LogWriter
	meterProvider  *sdkmetric.MeterProvider
	requestCount   otelmetric.Int64Counter
	requestLatency otelmetric.Float64Histogram
	sentryEnabled  bool
}

type otelProviders struct {
	traces         *sdktrace.TracerProvider
	logs           *sdklog.LoggerProvider
	logWriter      *LogWriter
	metrics        *sdkmetric.MeterProvider
	requestCount   otelmetric.Int64Counter
	requestLatency otelmetric.Float64Histogram
}

// New builds the enabled observability clients. Disabled integrations are
// skipped without opening network connections.
func New(d Dependency) (*Telemetry, error) {
	t := &Telemetry{}
	if d.Config.OpenObserve.Enabled {
		providers, err := newOTel(d.Config.OpenObserve)
		if err != nil {
			return nil, fmt.Errorf("configure openobserve: %w", err)
		}
		t.tracerProvider = providers.traces
		t.loggerProvider = providers.logs
		t.logWriter = providers.logWriter
		t.meterProvider = providers.metrics
		t.requestCount = providers.requestCount
		t.requestLatency = providers.requestLatency
		t.tracer = providers.traces.Tracer("kinstack/http")
	}
	if d.Config.GlitchTip.Enabled {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              d.Config.GlitchTip.DSN,
			Environment:      d.Config.GlitchTip.Environment,
			Release:          d.Config.GlitchTip.Release,
			Debug:            d.Config.GlitchTip.Debug,
			TracesSampleRate: d.Config.GlitchTip.TracesSampleRate,
			AttachStacktrace: true,
			DataCollection: &sentry.DataCollection{
				UserInfo:   sentry.Set(false),
				HTTPBodies: []sentry.BodyType{},
			},
		})
		if err != nil {
			return nil, fmt.Errorf("configure glitchtip: %w", err)
		}
		t.sentryEnabled = true
	}
	return t, nil
}

// Enabled reports whether any observability backend is active.
func (t *Telemetry) Enabled() bool {
	return t != nil && (t.tracerProvider != nil || t.sentryEnabled)
}

// LogWriter returns an extra writer that ships zerolog JSON lines over OTLP.
// It returns nil when log export is disabled.
func (t *Telemetry) LogWriter() *LogWriter {
	if t == nil {
		return nil
	}
	return t.logWriter
}

// Middleware traces and measures every HTTP request with OpenTelemetry. It is a
// no-op when OpenObserve is disabled.
func (t *Telemetry) Middleware() fiber.Handler {
	if t == nil || t.tracer == nil {
		return func(c fiber.Ctx) error { return c.Next() }
	}
	return func(c fiber.Ctx) error {
		path := c.Path()
		if strings.HasPrefix(path, "/build/") || path == "/health" || path == "/ready" {
			return c.Next()
		}
		start := time.Now()
		ctx := otel.GetTextMapPropagator().Extract(c.Context(), headerCarrier{c: c})
		ctx, span := t.tracer.Start(
			ctx,
			"HTTP "+c.Method(),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.request.method", c.Method()),
				attribute.String("url.path", c.Path()),
				attribute.String("server.address", c.Hostname()),
				attribute.String("url.scheme", c.Protocol()),
			),
		)
		c.SetContext(ctx)
		c.Locals(traceIDLocal, span.SpanContext().TraceID().String())
		defer span.End()

		err := c.Next()

		status := c.Response().StatusCode()
		route := ""
		if matched := c.Route(); matched != nil {
			route = matched.Path
		}
		span.SetAttributes(attribute.Int("http.response.status_code", status))
		if route != "" {
			span.SetName(c.Method() + " " + route)
			span.SetAttributes(
				attribute.String("http.route", route),
				attribute.String("http.request.method_original", c.Method()),
			)
		}
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else if status >= fiber.StatusInternalServerError {
			span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", status))
		}

		if t.requestCount != nil {
			attrs := otelmetric.WithAttributes(
				attribute.String("http.request.method", c.Method()),
				attribute.String("http.route", route),
				attribute.Int("http.response.status_code", status),
			)
			t.requestCount.Add(c.Context(), 1, attrs)
			t.requestLatency.Record(c.Context(), time.Since(start).Seconds(), attrs)
		}
		return err
	}
}

// CaptureException reports an unexpected error to GlitchTip with the request
// scope needed to correlate it with logs and traces.
func (t *Telemetry) CaptureException(c fiber.Ctx, err error) {
	if t == nil || !t.sentryEnabled || err == nil {
		return
	}
	hub := sentry.CurrentHub().Clone()
	hub.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(sentry.LevelError)
		scope.SetTag("request_id", c.RequestID())
		scope.SetTag("http.method", c.Method())
		scope.SetTag("http.path", c.Path())
		if route := c.Route(); route != nil && route.Path != "" {
			scope.SetTag("http.route", route.Path)
		}
		if traceID, ok := c.Locals(traceIDLocal).(string); ok && traceID != "" {
			scope.SetTag("trace_id", traceID)
		}
		hub.CaptureException(err)
	})
}

// Shutdown flushes pending metrics, logs, traces, and error events. It is safe
// to call on a nil Telemetry.
func (t *Telemetry) Shutdown(ctx context.Context) error {
	if t == nil {
		return nil
	}
	var errs []error
	if t.logWriter != nil {
		t.logWriter.Close()
	}
	if t.meterProvider != nil {
		errs = append(errs, t.meterProvider.Shutdown(ctx))
	}
	if t.loggerProvider != nil {
		errs = append(errs, t.loggerProvider.Shutdown(ctx))
	}
	if t.tracerProvider != nil {
		errs = append(errs, t.tracerProvider.Shutdown(ctx))
	}
	if t.sentryEnabled {
		sentry.Flush(2 * time.Second)
	}
	return errors.Join(errs...)
}

func newOTel(cfg config.OpenObserveConfig) (*otelProviders, error) {
	ctx := context.Background()
	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", cfg.ServiceName),
		attribute.String("service.version", cfg.Release),
		attribute.String("deployment.environment", cfg.Environment),
	))
	if err != nil {
		return nil, fmt.Errorf("build resource: %w", err)
	}

	headers := otlpHeaders(cfg)
	traceOptions := []otlptracehttp.Option{
		otlptracehttp.WithEndpointURL(cfg.Endpoint + "/api/" + cfg.Organization + "/v1/traces"),
		otlptracehttp.WithHeaders(headers),
	}
	if cfg.Insecure {
		traceOptions = append(traceOptions, otlptracehttp.WithInsecure())
	}
	traceExporter, err := otlptracehttp.New(ctx, traceOptions...)
	if err != nil {
		return nil, fmt.Errorf("create trace exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithSampler(
			sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.TracesSampleRate)),
		),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	providers := &otelProviders{traces: tracerProvider}
	if cfg.ExportLogs {
		providers.logs, providers.logWriter, err = newLogs(ctx, cfg, res, headers)
		if err != nil {
			return nil, err
		}
	}
	if cfg.ExportMetrics {
		providers.metrics, providers.requestCount, providers.requestLatency, err = newMetrics(
			ctx,
			cfg,
			res,
			headers,
		)
		if err != nil {
			return nil, err
		}
	}
	return providers, nil
}

func newLogs(
	ctx context.Context,
	cfg config.OpenObserveConfig,
	res *resource.Resource,
	headers map[string]string,
) (*sdklog.LoggerProvider, *LogWriter, error) {
	options := []otlploghttp.Option{
		otlploghttp.WithEndpointURL(cfg.Endpoint + "/api/" + cfg.Organization + "/v1/logs"),
		otlploghttp.WithHeaders(headers),
	}
	if cfg.Insecure {
		options = append(options, otlploghttp.WithInsecure())
	}
	exporter, err := otlploghttp.New(ctx, options...)
	if err != nil {
		return nil, nil, fmt.Errorf("create log exporter: %w", err)
	}
	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)
	return provider, NewLogWriter(provider.Logger("kinstack")), nil
}

func newMetrics(
	ctx context.Context,
	cfg config.OpenObserveConfig,
	res *resource.Resource,
	headers map[string]string,
) (*sdkmetric.MeterProvider, otelmetric.Int64Counter, otelmetric.Float64Histogram, error) {
	options := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpointURL(cfg.Endpoint + "/api/" + cfg.Organization + "/v1/metrics"),
		otlpmetrichttp.WithHeaders(headers),
	}
	if cfg.Insecure {
		options = append(options, otlpmetrichttp.WithInsecure())
	}
	exporter, err := otlpmetrichttp.New(ctx, options...)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create metric exporter: %w", err)
	}
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter)),
	)
	otel.SetMeterProvider(provider)

	meter := provider.Meter("kinstack/http")
	count, err := meter.Int64Counter(
		"http.server.requests",
		otelmetric.WithDescription("Total number of HTTP server requests."),
		otelmetric.WithUnit("{request}"),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create request counter: %w", err)
	}
	latency, err := meter.Float64Histogram(
		"http.server.request.duration",
		otelmetric.WithDescription("HTTP server request latency."),
		otelmetric.WithUnit("s"),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create request latency histogram: %w", err)
	}
	return provider, count, latency, nil
}

func otlpHeaders(cfg config.OpenObserveConfig) map[string]string {
	headers := map[string]string{"stream-name": cfg.StreamName}
	if cfg.AuthHeader != "" {
		headers["Authorization"] = cfg.AuthHeader
	}
	return headers
}
