package utils

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestWithLogging(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		remoteAddr string
	}{
		{
			name:       "log GET request",
			method:     "GET",
			path:       "/test-path",
			remoteAddr: "192.168.1.1:1234",
		},
		{
			name:       "log POST request",
			method:     "POST",
			path:       "/api/data",
			remoteAddr: "10.0.0.1:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Redirect log output to a buffer
			var buf bytes.Buffer
			log.SetOutput(&buf)
			defer log.SetOutput(os.Stderr) // Reset after test

			// Simple handler that just returns 200 OK
			innerHandler := func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}

			// Wrap the handler with middleware
			handler := WithLogging(innerHandler)

			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.RemoteAddr = tt.remoteAddr
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Verify status code
			if rr.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rr.Code)
			}

			// Verify log output
			logOutput := buf.String()
			if !strings.Contains(logOutput, "METHOD="+tt.method) {
				t.Errorf("log missing method: expected %s, got %s", tt.method, logOutput)
			}
			if !strings.Contains(logOutput, "PATH="+tt.path) {
				t.Errorf("log missing path: expected %s, got %s", tt.path, logOutput)
			}
			if !strings.Contains(logOutput, "REMOTE="+tt.remoteAddr) {
				t.Errorf("log missing remote addr: expected %s, got %s", tt.remoteAddr, logOutput)
			}
			if !strings.Contains(logOutput, "DURATION=") {
				t.Errorf("log missing duration: %s", logOutput)
			}
		})
	}
}
