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

		// Wrap the ResponseWriter to capture the status code
		lrw := newLoggingResponseWriter(w)

		next(lrw, r)

		slog.Info("request_processed",
			"http_method", r.Method,
			"path", r.URL.Path,
			"remote_ip", r.RemoteAddr,
			"status", lrw.statusCode,
			"duration", time.Since(start).String(),
		)
	}
}

// loggingResponseWriter wraps http.ResponseWriter to capture the status code.
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
