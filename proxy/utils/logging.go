package utils

import (
	"log/slog"
	"net/http"
	"time"
)

// WithLogging wraps an http.HandlerFunc to log request details
func WithLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// We could wrap ResponseWriter to capture status code, but for now we keep it simple
		next(w, r)

		slog.Info("request_processed",
			"http_method", r.Method,
			"path", r.URL.Path,
			"remote_ip", r.RemoteAddr,
			"duration", time.Since(start).String(),
		)
	}
}
