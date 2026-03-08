package db

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// SetupTestTracer initializes a test tracer provider and returns an exporter
// to verify spans in unit tests.
func SetupTestTracer() *tracetest.InMemoryExporter {
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSampler(trace.AlwaysSample()),
		trace.WithSpanProcessor(trace.NewSimpleSpanProcessor(exporter)),
	)
	otel.SetTracerProvider(tp)
	return exporter
}

// GetSpanAttributes returns a map of attributes for the first span in the exporter.
func GetSpanAttributes(exporter *tracetest.InMemoryExporter) map[string]string {
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		return nil
	}

	attrs := make(map[string]string)
	for _, attr := range spans[0].Attributes {
		attrs[string(attr.Key)] = attr.Value.AsString()
	}
	return attrs
}
