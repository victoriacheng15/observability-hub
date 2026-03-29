package telemetry

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	os.Setenv("APP_ENV", "test")
	SilenceLogs()
	os.Exit(m.Run())
}

func TestInit(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		endpoint string
		service  string
	}{
		{
			name:     "Disabled when endpoint missing",
			endpoint: "",
			service:  "test",
		},
		{
			name:     "Uses default service name",
			endpoint: "localhost:4317",
			service:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.endpoint != "" {
				os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", tt.endpoint)
			} else {
				os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
			}
			defer os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

			shutdown, err := Init(ctx, tt.service)
			if err != nil {
				t.Fatalf("Init failed: %v", err)
			}
			if shutdown == nil {
				t.Fatal("shutdown function should not be nil")
			}
			shutdown()
		})
	}
}

func TestAttributes(t *testing.T) {
	tests := []struct {
		name string
		attr Attribute
		want interface{}
	}{
		{
			name: "StringAttribute",
			attr: StringAttribute("key", "value"),
			want: "value",
		},
		{
			name: "IntAttribute",
			attr: IntAttribute("key", 123),
			want: int64(123),
		},
		{
			name: "BoolAttribute",
			attr: BoolAttribute("key", true),
			want: true,
		},
		{
			name: "Int64Attribute",
			attr: Int64Attribute("key", int64(456)),
			want: int64(456),
		},
		{
			name: "Float64Attribute",
			attr: Float64Attribute("key", 1.23),
			want: 1.23,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch v := tt.want.(type) {
			case string:
				if tt.attr.Value.AsString() != v {
					t.Errorf("got %v, want %v", tt.attr.Value.AsString(), v)
				}
			case int64:
				if tt.attr.Value.AsInt64() != v {
					t.Errorf("got %v, want %v", tt.attr.Value.AsInt64(), v)
				}
			case bool:
				if tt.attr.Value.AsBool() != v {
					t.Errorf("got %v, want %v", tt.attr.Value.AsBool(), v)
				}
			case float64:
				if tt.attr.Value.AsFloat64() != v {
					t.Errorf("got %v, want %v", tt.attr.Value.AsFloat64(), v)
				}
			}
		})
	}
}

func TestLogs(t *testing.T) {
	tests := []struct {
		name   string
		testFn func(t *testing.T)
	}{
		{
			name: "GetLogger",
			testFn: func(t *testing.T) {
				l := GetLogger("test-logger")
				if l == nil {
					t.Fatal("Logger should not be nil")
				}
				lDefault := GetLogger("")
				if lDefault == nil {
					t.Fatal("Default logger should not be nil")
				}
			},
		},
		{
			name: "Log Methods",
			testFn: func(t *testing.T) {
				Info("info message", "key", "val")
				Warn("warn message", "key", "val")
				Error("error message", "key", "val")
			},
		},
		{
			name: "PII Masking",
			testFn: func(t *testing.T) {
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
			},
		},
		{
			name: "PIIHandler",
			testFn: func(t *testing.T) {
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
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFn(t)
		})
	}
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

	tests := []struct {
		name   string
		testFn func(t *testing.T)
	}{
		{
			name: "Counter",
			testFn: func(t *testing.T) {
				c, err := NewInt64Counter(meter, "test-counter", "desc")
				if err != nil {
					t.Fatalf("Failed to create counter: %v", err)
				}
				AddInt64Counter(ctx, c, 1, StringAttribute("tag", "val"))
			},
		},
		{
			name: "Histogram",
			testFn: func(t *testing.T) {
				h, err := NewInt64Histogram(meter, "test-histogram", "desc", "ms")
				if err != nil {
					t.Fatalf("Failed to create histogram: %v", err)
				}
				RecordInt64Histogram(ctx, h, 100, StringAttribute("tag", "val"))
			},
		},
		{
			name: "Gauges",
			testFn: func(t *testing.T) {
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
			},
		},
		{
			name: "WithMetricAttributes",
			testFn: func(t *testing.T) {
				opt := WithMetricAttributes(StringAttribute("a", "b"))
				if opt == nil {
					t.Error("Expected non-nil option")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFn(t)
		})
	}
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

	tests := []struct {
		name   string
		testFn func(t *testing.T)
	}{
		{
			name: "SpanFromContext",
			testFn: func(t *testing.T) {
				s := SpanFromContext(ctx)
				if s == nil {
					t.Fatal("Span should not be nil")
				}
			},
		},
		{
			name: "Trace Options",
			testFn: func(t *testing.T) {
				WithAttributes(StringAttribute("a", "b"))
				WithEventAttributes(StringAttribute("e", "f"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFn(t)
		})
	}
}

func TestHTTPHandler(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		path        string
		wantStatus  int
	}{
		{
			name:        "Basic instrumentation",
			serviceName: "test-service",
			path:        "/test",
			wantStatus:  http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.wantStatus)
			})
			instrumented := NewHTTPHandler(handler, tt.serviceName)

			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()
			instrumented.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, w.Code)
			}
		})
	}
}

type countingHandler struct {
	enabled   bool
	handled   int
	withAttrs int
	withGroup int
}

func (h *countingHandler) Enabled(context.Context, slog.Level) bool { return h.enabled }
func (h *countingHandler) Handle(context.Context, slog.Record) error {
	h.handled++
	return nil
}
func (h *countingHandler) WithAttrs([]slog.Attr) slog.Handler {
	h.withAttrs++
	return h
}
func (h *countingHandler) WithGroup(string) slog.Handler {
	h.withGroup++
	return h
}

func TestMultiHandler(t *testing.T) {
	h1 := &countingHandler{enabled: false}
	h2 := &countingHandler{enabled: true}

	m := &MultiHandler{Handlers: []slog.Handler{h1, h2}}
	if !m.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("expected Enabled to be true when any handler is enabled")
	}

	if err := m.Handle(context.Background(), slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)); err != nil {
		t.Fatalf("handle error: %v", err)
	}
	if h1.handled != 1 || h2.handled != 1 {
		t.Fatalf("expected both handlers to be invoked, got h1=%d h2=%d", h1.handled, h2.handled)
	}

	_ = m.WithAttrs([]slog.Attr{slog.String("k", "v")})
	_ = m.WithGroup("g")
	if h1.withAttrs != 1 || h2.withAttrs != 1 {
		t.Fatalf("expected WithAttrs to be forwarded, got h1=%d h2=%d", h1.withAttrs, h2.withAttrs)
	}
	if h1.withGroup != 1 || h2.withGroup != 1 {
		t.Fatalf("expected WithGroup to be forwarded, got h1=%d h2=%d", h1.withGroup, h2.withGroup)
	}
}

func TestInit_EndpointMissing_DisablesTelemetry(t *testing.T) {
	oldEnv := os.Getenv("APP_ENV")
	oldEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	defer func() {
		os.Setenv("APP_ENV", oldEnv)
		if oldEndpoint == "" {
			os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
		} else {
			os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", oldEndpoint)
		}
		SilenceLogs()
	}()

	os.Setenv("APP_ENV", "dev")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")

	shutdown, err := Init(context.Background(), "svc")
	if err != nil {
		t.Fatalf("Init error: %v", err)
	}
	if shutdown == nil {
		t.Fatal("expected non-nil shutdown")
	}
	shutdown()
}
