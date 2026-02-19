package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	otellog "go.opentelemetry.io/otel/log"
	otellogglobal "go.opentelemetry.io/otel/log/global"
	metricapi "go.opentelemetry.io/otel/metric"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Re-export common OTel types and functions to centralize dependency management
type (
	Span           = trace.Span
	Tracer         = trace.Tracer
	Attribute      = attribute.KeyValue
	Code           = codes.Code
	MeterProvider  = metricapi.MeterProvider
	Meter          = metricapi.Meter
	Int64Counter   = metricapi.Int64Counter
	Int64Histogram = metricapi.Int64Histogram
	LoggerProvider = otellog.LoggerProvider
	Logger         = otellog.Logger
)

const (
	CodeError = codes.Error
	CodeOk    = codes.Ok
	ScopeName = "observability-hub"
)

// GetTracer returns a tracer with the provided name.
func GetTracer(name string) Tracer {
	return otel.Tracer(name)
}

// GetMeter returns a meter with the provided name.
func GetMeter(name string) Meter {
	if name == "" {
		name = ScopeName
	}
	return otel.Meter(name)
}

// GetLogger returns an OpenTelemetry logger with the provided name.
func GetLogger(name string) Logger {
	if name == "" {
		name = ScopeName
	}
	return otellogglobal.GetLoggerProvider().Logger(name)
}

// NewInt64Counter creates an int64 counter with an optional description.
func NewInt64Counter(meter Meter, name, description string) (Int64Counter, error) {
	opts := []metricapi.Int64CounterOption{}
	if description != "" {
		opts = append(opts, metricapi.WithDescription(description))
	}
	return meter.Int64Counter(name, opts...)
}

// NewInt64Histogram creates an int64 histogram with optional description and unit.
func NewInt64Histogram(meter Meter, name, description, unit string) (Int64Histogram, error) {
	opts := []metricapi.Int64HistogramOption{}
	if description != "" {
		opts = append(opts, metricapi.WithDescription(description))
	}
	if unit != "" {
		opts = append(opts, metricapi.WithUnit(unit))
	}
	return meter.Int64Histogram(name, opts...)
}

// AddInt64Counter adds a value to an int64 counter with optional attributes.
func AddInt64Counter(ctx context.Context, counter Int64Counter, value int64, attrs ...Attribute) {
	counter.Add(ctx, value, metricapi.WithAttributes(attrs...))
}

// RecordInt64Histogram records a value in an int64 histogram with optional attributes.
func RecordInt64Histogram(ctx context.Context, histogram Int64Histogram, value int64, attrs ...Attribute) {
	histogram.Record(ctx, value, metricapi.WithAttributes(attrs...))
}

// SpanFromContext returns the current span from the context.
func SpanFromContext(ctx context.Context) Span {
	return trace.SpanFromContext(ctx)
}

// WithAttributes returns a SpanStartOption that sets the provided attributes.
func WithAttributes(attrs ...Attribute) trace.SpanStartOption {
	return trace.WithAttributes(attrs...)
}

// WithEventAttributes returns an EventOption that sets the provided attributes.
func WithEventAttributes(attrs ...Attribute) trace.EventOption {
	return trace.WithAttributes(attrs...)
}

// StringAttribute creates a new string attribute.
func StringAttribute(key, value string) Attribute {
	return attribute.String(key, value)
}

// IntAttribute creates a new integer attribute.
func IntAttribute(key string, value int) Attribute {
	return attribute.Int(key, value)
}

// BoolAttribute creates a new boolean attribute.
func BoolAttribute(key string, value bool) Attribute {
	return attribute.Bool(key, value)
}

// Init initializes OpenTelemetry trace, metric and log providers over OTLP gRPC.
// It reads OTEL_EXPORTER_OTLP_ENDPOINT from the environment.
func Init(
	ctx context.Context,
	serviceName string,
) (shutdownTracer func(context.Context) error, shutdownMeter func(context.Context) error, shutdownLogger func(context.Context) error, err error) {
	shutdownTracer = func(context.Context) error { return nil }
	shutdownMeter = func(context.Context) error { return nil }
	shutdownLogger = func(context.Context) error { return nil }

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		slog.Warn("OTEL_EXPORTER_OTLP_ENDPOINT not set, telemetry disabled")
		return shutdownTracer, shutdownMeter, shutdownLogger, nil
	}

	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		err = fmt.Errorf("failed to create gRPC connection to collector: %w", err)
		return
	}

	if serviceName == "" {
		serviceName = "unknown-service"
	}

	hostname, _ := os.Hostname()
	res := resource.NewSchemaless(
		semconv.ServiceName(serviceName),
		semconv.HostName(hostname),
	)

	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		err = fmt.Errorf("failed to create trace exporter: %w", err)
		return
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	shutdownTracer = tp.Shutdown

	metricExporter, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithGRPCConn(conn))
	if err != nil {
		err = fmt.Errorf("failed to create metric exporter: %w", err)
		return
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(3*time.Second)),
		),
		sdkmetric.WithResource(res),
	)
	otel.SetMeterProvider(mp)
	shutdownMeter = mp.Shutdown

	logExporter, err := otlploggrpc.New(ctx, otlploggrpc.WithGRPCConn(conn))
	if err != nil {
		err = fmt.Errorf("failed to create log exporter: %w", err)
		return
	}

	consoleExporter, err := stdoutlog.New()
	if err != nil {
		err = fmt.Errorf("failed to create stdout log exporter: %w", err)
		return
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithProcessor(sdklog.NewSimpleProcessor(consoleExporter)),
		sdklog.WithResource(res),
	)
	otellogglobal.SetLoggerProvider(lp)
	shutdownLogger = lp.Shutdown

	slog.SetDefault(slog.New(otelslog.NewHandler(ScopeName, otelslog.WithLoggerProvider(lp))))
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(e error) {
		slog.Error("otel_error", "error", e)
	}))

	slog.Info("otel_telemetry_enabled", "endpoint", endpoint, "service", serviceName)
	return
}

// NewHTTPHandler wraps an http.Handler with OpenTelemetry instrumentation.
func NewHTTPHandler(h http.Handler, serviceName string) http.Handler {
	return otelhttp.NewHandler(h, serviceName,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		}),
	)
}
