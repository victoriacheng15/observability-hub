package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestSetup(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		logMsg      string
		attrs       []slog.Attr
	}{
		{
			name:        "Basic log entry",
			serviceName: "test-service",
			logMsg:      "hello world",
		},
		{
			name:        "Log with extra attributes",
			serviceName: "metrics-app",
			logMsg:      "data point",
			attrs: []slog.Attr{
				slog.String("component", "collector"),
				slog.Int("count", 42),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			Setup(&buf, tt.serviceName)

			// Log the message with attributes
			args := make([]any, 0, len(tt.attrs)*2)
			for _, attr := range tt.attrs {
				args = append(args, attr.Key, attr.Value.Any())
			}
			slog.Info(tt.logMsg, args...)

			// Verify JSON output
			var logEntry map[string]any
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				t.Fatalf("Failed to parse log JSON: %v", err)
			}

			// Assertions
			if logEntry["msg"] != tt.logMsg {
				t.Errorf("Expected msg %q, got %q", tt.logMsg, logEntry["msg"])
			}
			if logEntry["service"] != tt.serviceName {
				t.Errorf("Expected service %q, got %q", tt.serviceName, logEntry["service"])
			}
			if logEntry["level"] != "INFO" {
				t.Errorf("Expected level INFO, got %v", logEntry["level"])
			}

			// Check extra attributes
			for _, attr := range tt.attrs {
				val, ok := logEntry[attr.Key]
				if !ok {
					t.Errorf("Missing expected attribute %q", attr.Key)
					continue
				}
				// json.Unmarshal converts numbers to float64 by default
				if attr.Value.Kind() == slog.KindInt64 {
					if int(val.(float64)) != int(attr.Value.Int64()) {
						t.Errorf("Attribute %q: expected %v, got %v", attr.Key, attr.Value, val)
					}
				} else if val != attr.Value.Any() {
					t.Errorf("Attribute %q: expected %v, got %v", attr.Key, attr.Value, val)
				}
			}
		})
	}
}
