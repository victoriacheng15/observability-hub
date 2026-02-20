package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

const (
	CodeError = codes.Error
	CodeOk    = codes.Ok
)

// GetTracer returns a tracer with the provided name.
func GetTracer(name string) trace.Tracer {
	if name == "" {
		name = ScopeName
	}
	return otel.Tracer(name)
}

// SpanFromContext returns the current span from the context.
func SpanFromContext(ctx context.Context) trace.Span {
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

func initTraces(ctx context.Context, conn *grpc.ClientConn, res *resource.Resource) (func(context.Context) error, error) {
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
