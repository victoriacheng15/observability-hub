package logger

import (
	"log/slog"
	"os"
)

// Setup initializes the global slog logger to output JSON to stdout.
// It adds a permanent "service" field to all log entries.
func Setup(serviceName string) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts).
		WithAttrs([]slog.Attr{
			slog.String("service", serviceName),
		})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
