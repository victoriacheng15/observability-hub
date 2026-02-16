package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Re-export common OTel types and functions to centralize dependency management
type (
	Span      = trace.Span
	Tracer    = trace.Tracer
	Attribute = attribute.KeyValue
	Code      = codes.Code
)

const (
	CodeError = codes.Error
	CodeOk    = codes.Ok
)

// GetTracer returns a tracer with the provided name.
func GetTracer(name string) Tracer {
	return otel.Tracer(name)
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

// Init initializes the OpenTelemetry TracerProvider with an OTLP gRPC exporter.
// It reads OTEL_EXPORTER_OTLP_ENDPOINT from the environment (required).
// Returns a shutdown function that flushes and closes the exporter.
func Init(ctx context.Context) (func(context.Context) error, error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		slog.Warn("OTEL_EXPORTER_OTLP_ENDPOINT not set, tracing disabled")
		return func(context.Context) error { return nil }, nil
	}

	conn, err := grpc.NewClient(endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	exporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "unknown-service"
	}

	res := resource.NewSchemaless(
		semconv.ServiceName(serviceName),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	slog.Info("otel_tracing_enabled", "endpoint", endpoint, "service", serviceName)

	return tp.Shutdown, nil
}

// NewHTTPHandler wraps an http.Handler with OpenTelemetry instrumentation.
func NewHTTPHandler(h http.Handler, serviceName string) http.Handler {
	return otelhttp.NewHandler(h, serviceName,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		}),
	)
}
