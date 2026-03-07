package telemetry

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
}

func TestAttributes(t *testing.T) {
	t.Run("StringAttribute", func(t *testing.T) {
		attr := StringAttribute("key", "value")
		if attr.Key != "key" || attr.Value.AsString() != "value" {
			t.Errorf("Unexpected attribute: %v", attr)
		}
	})
	t.Run("IntAttribute", func(t *testing.T) {
		attr := IntAttribute("key", 123)
		if attr.Key != "key" || attr.Value.AsInt64() != 123 {
			t.Errorf("Unexpected attribute: %v", attr)
		}
	})
	t.Run("BoolAttribute", func(t *testing.T) {
		attr := BoolAttribute("key", true)
		if attr.Key != "key" || !attr.Value.AsBool() {
			t.Errorf("Unexpected attribute: %v", attr)
		}
	})
	t.Run("Int64Attribute", func(t *testing.T) {
		attr := Int64Attribute("key", int64(456))
		if attr.Key != "key" || attr.Value.AsInt64() != 456 {
			t.Errorf("Unexpected attribute: %v", attr)
		}
	})
	t.Run("Float64Attribute", func(t *testing.T) {
		attr := Float64Attribute("key", 1.23)
		if attr.Key != "key" || attr.Value.AsFloat64() != 1.23 {
			t.Errorf("Unexpected attribute: %v", attr)
		}
	})
}

func TestLogs(t *testing.T) {
	t.Run("GetLogger", func(t *testing.T) {
		l := GetLogger("test-logger")
		if l == nil {
			t.Fatal("Logger should not be nil")
		}
		lDefault := GetLogger("")
		if lDefault == nil {
			t.Fatal("Default logger should not be nil")
		}
	})

	t.Run("Log Methods", func(t *testing.T) {
		Info("info message", "key", "val")
		Warn("warn message", "key", "val")
		Error("error message", "key", "val")
	})

	t.Run("PII Masking", func(t *testing.T) {
		attr := MaskPII(nil, slog.String("password", "secret123"))
		if attr.Value.String() != "[REDACTED]" {
			t.Errorf("Expected [REDACTED], got %v", attr.Value)
		}

		attrEmail := MaskPII(nil, slog.String("email", "test@example.com"))
		if attrEmail.Value.String() != "[REDACTED]" {
			t.Errorf("Expected [REDACTED], got %v", attrEmail.Value)
		}

		attrSafe := MaskPII(nil, slog.String("safe", "data"))
		if attrSafe.Value.String() != "data" {
			t.Errorf("Expected data, got %v", attrSafe.Value)
		}

		// Nested group: sensitive key inside a slog.Group must be redacted
		attrGroup := MaskPII(nil, slog.Group("request", slog.String("token", "nested-secret"), slog.String("path", "/api")))
		for _, ga := range attrGroup.Value.Group() {
			if ga.Key == "token" && ga.Value.String() != "[REDACTED]" {
				t.Errorf("Expected nested token to be [REDACTED], got %v", ga.Value)
			}
			if ga.Key == "path" && ga.Value.String() != "/api" {
				t.Errorf("Expected safe nested attr to pass through, got %v", ga.Value)
			}
		}
	})

	t.Run("PIIHandler", func(t *testing.T) {
		newHandler := func(buf *strings.Builder) slog.Handler {
			return NewPIIHandler(slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
		}

		t.Run("Handle redacts top-level sensitive attr", func(t *testing.T) {
			var buf strings.Builder
			slog.New(newHandler(&buf)).Info("test", "token", "top-secret")
			out := buf.String()
			if strings.Contains(out, "top-secret") {
				t.Error("expected token value to be redacted")
			}
			if !strings.Contains(out, "[REDACTED]") {
				t.Error("expected [REDACTED] in output")
			}
		})

		t.Run("WithAttrs redacts sensitive attrs", func(t *testing.T) {
			var buf strings.Builder
			slog.New(newHandler(&buf)).With("password", "mypassword").Info("test")
			if strings.Contains(buf.String(), "mypassword") {
				t.Error("expected password to be redacted in WithAttrs")
			}
		})

		t.Run("Handle redacts nested group sensitive attr", func(t *testing.T) {
			var buf strings.Builder
			slog.New(newHandler(&buf)).Info("test", slog.Group("req", "token", "nested-secret", "path", "/api"))
			out := buf.String()
			if strings.Contains(out, "nested-secret") {
				t.Error("expected nested token to be redacted")
			}
			if !strings.Contains(out, "/api") {
				t.Error("expected safe nested attr to pass through")
			}
		})

		t.Run("safe attrs pass through", func(t *testing.T) {
			var buf strings.Builder
			slog.New(newHandler(&buf)).Info("test", "user_id", "12345")
			if !strings.Contains(buf.String(), "12345") {
				t.Error("expected safe attr to pass through unmodified")
			}
		})
	})
}

func TestMetrics(t *testing.T) {
	meter := GetMeter("test-meter")
	if meter == nil {
		t.Fatal("Meter should not be nil")
	}
	meterDefault := GetMeter("")
	if meterDefault == nil {
		t.Fatal("Default meter should not be nil")
	}

	ctx := context.Background()

	t.Run("Counter", func(t *testing.T) {
		c, err := NewInt64Counter(meter, "test-counter", "desc")
		if err != nil {
			t.Fatalf("Failed to create counter: %v", err)
		}
		AddInt64Counter(ctx, c, 1, StringAttribute("tag", "val"))
	})

	t.Run("Histogram", func(t *testing.T) {
		h, err := NewInt64Histogram(meter, "test-histogram", "desc", "ms")
		if err != nil {
			t.Fatalf("Failed to create histogram: %v", err)
		}
		RecordInt64Histogram(ctx, h, 100, StringAttribute("tag", "val"))
	})

	t.Run("Gauges", func(t *testing.T) {
		_, err := NewInt64ObservableGauge(meter, "test-gauge-int", "desc", func(ctx context.Context, obs Int64Observer) error {
			obs.Observe(1)
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to create int gauge: %v", err)
		}

		_, err = NewFloat64ObservableGauge(meter, "test-gauge-float", "desc", func(ctx context.Context, obs Float64Observer) error {
			obs.Observe(1.5)
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to create float gauge: %v", err)
		}
	})

	t.Run("WithMetricAttributes", func(t *testing.T) {
		opt := WithMetricAttributes(StringAttribute("a", "b"))
		if opt == nil {
			t.Error("Expected non-nil option")
		}
	})
}

func TestTraces(t *testing.T) {
	tracer := GetTracer("test-tracer")
	if tracer == nil {
		t.Fatal("Tracer should not be nil")
	}
	tracerDefault := GetTracer("")
	if tracerDefault == nil {
		t.Fatal("Default tracer should not be nil")
	}

	ctx := context.Background()
	ctx, span := tracer.Start(ctx, "test-span")
	defer span.End()

	t.Run("SpanFromContext", func(t *testing.T) {
		s := SpanFromContext(ctx)
		if s == nil {
			t.Fatal("Span should not be nil")
		}
	})

	t.Run("Trace Options", func(t *testing.T) {
		WithAttributes(StringAttribute("a", "b"))
		WithEventAttributes(StringAttribute("e", "f"))
	})
}

func TestHTTPHandler(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	instrumented := NewHTTPHandler(handler, "test-service")

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	instrumented.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
