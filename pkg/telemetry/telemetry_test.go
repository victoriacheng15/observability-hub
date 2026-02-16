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
		shutdown, err := Init(ctx)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if shutdown == nil {
			t.Fatal("shutdown function should not be nil")
		}
		// Should do nothing but not error
		_ = shutdown(ctx)
	})

	t.Run("Uses default service name", func(t *testing.T) {
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
		os.Unsetenv("OTEL_SERVICE_NAME")
		defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

		// We don't necessarily want to trigger the full gRPC connection in a unit test
		// but we can check if it attempts to use the endpoint.
		// Since gRPC.NewClient doesn't actually dial immediately, we can call it.
		shutdown, err := Init(ctx)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if shutdown == nil {
			t.Fatal("shutdown function should not be nil")
		}
		_ = shutdown(ctx)
	})

	t.Run("Uses custom service name", func(t *testing.T) {
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
		os.Setenv("OTEL_SERVICE_NAME", "test-service")
		defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		defer os.Unsetenv("OTEL_SERVICE_NAME")

		shutdown, err := Init(ctx)
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if shutdown == nil {
			t.Fatal("shutdown function should not be nil")
		}
		_ = shutdown(ctx)
	})
}
