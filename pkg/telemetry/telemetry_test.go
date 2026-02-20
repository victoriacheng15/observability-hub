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
		shutdown, err := Init(ctx, "test")
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if shutdown == nil {
			t.Fatal("shutdown function should not be nil")
		}
		shutdown()
	})

	t.Run("Uses default service name", func(t *testing.T) {
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
		defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

		shutdown, err := Init(ctx, "")
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if shutdown == nil {
			t.Fatal("shutdown function should not be nil")
		}
		shutdown()
	})

	t.Run("Uses custom service name", func(t *testing.T) {
		os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
		defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

		shutdown, err := Init(ctx, "test-service")
		if err != nil {
			t.Fatalf("Init failed: %v", err)
		}
		if shutdown == nil {
			t.Fatal("shutdown function should not be nil")
		}
		shutdown()
	})
}
