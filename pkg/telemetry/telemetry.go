package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	metricapi "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Re-export common OTel types to centralize dependency management
type (
	Span            = trace.Span
	Tracer          = trace.Tracer
	Attribute       = attribute.KeyValue
	Code            = codes.Code
	MeterProvider   = metricapi.MeterProvider
	Meter           = metricapi.Meter
	Int64Counter    = metricapi.Int64Counter
	Int64Histogram  = metricapi.Int64Histogram
	Int64Observer   = metricapi.Int64Observer
	Float64Observer = metricapi.Float64Observer
	LoggerProvider  = otellog.LoggerProvider
	Logger          = otellog.Logger
)

const (
	ScopeName = "observability-hub"
)

// Init initializes OpenTelemetry trace, metric and log providers over OTLP gRPC.
func Init(ctx context.Context, serviceName string) (func(), error) {
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		slog.Warn("OTEL_EXPORTER_OTLP_ENDPOINT not set, telemetry disabled")
		return func() {}, nil
	}

	conn, err := grpc.NewClient(
		endpoint,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}

	if serviceName == "" {
		serviceName = "unknown-service"
	}

	hostname, _ := os.Hostname()
	res := resource.NewSchemaless(
		semconv.ServiceName(serviceName),
		semconv.HostName(hostname),
	)

	shutdownTracer, err := initTraces(ctx, conn, res)
	if err != nil {
		conn.Close()
		return nil, err
	}

	shutdownMeter, err := initMetrics(ctx, conn, res)
	if err != nil {
		_ = shutdownTracer(ctx)
		conn.Close()
		return nil, err
	}

	shutdownLogger, err := initLogs(ctx, conn, res)
	if err != nil {
		_ = shutdownMeter(ctx)
		_ = shutdownTracer(ctx)
		conn.Close()
		return nil, err
	}

	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(e error) {
		slog.Error("otel_error", "error", e)
	}))

	slog.Info("otel_telemetry_enabled", "endpoint", endpoint, "service", serviceName)

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if shutdownTracer != nil {
			if err := shutdownTracer(shutdownCtx); err != nil {
				Error("otel_shutdown_failed", "component", "tracer", "error", err)
			}
		}
		if shutdownMeter != nil {
			if err := shutdownMeter(shutdownCtx); err != nil {
				Error("otel_shutdown_failed", "component", "meter", "error", err)
			}
		}
		if shutdownLogger != nil {
			if err := shutdownLogger(shutdownCtx); err != nil {
				Error("otel_shutdown_failed", "component", "logger", "error", err)
			}
		}
		if conn != nil {
			conn.Close()
		}
	}, nil
}

// NewHTTPHandler wraps an http.Handler with OpenTelemetry instrumentation.
func NewHTTPHandler(h http.Handler, serviceName string) http.Handler {
	return otelhttp.NewHandler(h, serviceName,
		otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		}),
	)
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

// Int64Attribute creates a new 64-bit integer attribute.
func Int64Attribute(key string, value int64) Attribute {
	return attribute.Int64(key, value)
}

// Float64Attribute creates a new 64-bit float attribute.
func Float64Attribute(key string, value float64) Attribute {
	return attribute.Float64(key, value)
}
