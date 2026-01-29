package logger

import (
	"io"
	"log/slog"
)

// Setup initializes the global slog logger to output JSON to the provided writer.
// It adds a permanent "service" field to all log entries.
func Setup(w io.Writer, serviceName string) {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	handler := slog.NewJSONHandler(w, opts).
		WithAttrs([]slog.Attr{
			slog.String("service", serviceName),
		})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
