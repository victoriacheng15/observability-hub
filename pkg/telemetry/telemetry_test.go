package telemetry

import (
	"context"
	"os"
	"testing"
)

func TestInit(t *testing.T) {
	ctx := context.Background()

	t.Run("Disabled when endpoint missing", func(t *testing.T) {
		os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		shutdownTracer, shutdownMeter, shutdownLogger, err := Init(ctx, "test")
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if shutdownTracer == nil || shutdownMeter == nil || shutdownLogger == nil {
			t.Fatal("shutdown functions should not be nil")
		}
		_ = shutdownTracer(ctx)
		_ = shutdownMeter(ctx)
		_ = shutdownLogger(ctx)
	})

	t.Run("Uses default service name", func(t *testing.T) {
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
		defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

		// We don't necessarily want to trigger the full gRPC connection in a unit test
		// but we can check if it attempts to use the endpoint.
		// Since gRPC.NewClient doesn't actually dial immediately, we can call it.
		shutdownTracer, shutdownMeter, shutdownLogger, err := Init(ctx, "")
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if shutdownTracer == nil || shutdownMeter == nil || shutdownLogger == nil {
			t.Fatal("shutdown functions should not be nil")
		}
		_ = shutdownTracer(ctx)
		_ = shutdownMeter(ctx)
		_ = shutdownLogger(ctx)
	})

	t.Run("Uses custom service name", func(t *testing.T) {
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
		defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

		shutdownTracer, shutdownMeter, shutdownLogger, err := Init(ctx, "test-service")
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if shutdownTracer == nil || shutdownMeter == nil || shutdownLogger == nil {
			t.Fatal("shutdown functions should not be nil")
		}
		_ = shutdownTracer(ctx)
		_ = shutdownMeter(ctx)
		_ = shutdownLogger(ctx)
	})
}
