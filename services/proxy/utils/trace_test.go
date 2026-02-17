package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSyntheticTraceHandler(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		body        string
		trafficMode string
	}{
		{
			name:        "valid payload",
			path:        "/api/trace/synthetic/synth-123",
			body:        `{"region":"us-east-1","timezone":"UTC","device":"mobile","network_type":"5g"}`,
			trafficMode: "replay",
		},
		{
			name:        "invalid payload still succeeds",
			path:        "/api/trace/synthetic/synth-err",
			body:        `{"region":`,
			trafficMode: "synthetic",
		},
		{
			name:        "empty payload succeeds",
			path:        "/api/trace/synthetic/synth-empty",
			body:        "",
			trafficMode: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			if tt.trafficMode != "" {
				req.Header.Set("X-Traffic-Mode", tt.trafficMode)
			}
			w := httptest.NewRecorder()

			SyntheticTraceHandler(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", w.Code)
			}
			if got := w.Header().Get("Content-Type"); got != "application/json" {
				t.Fatalf("expected content-type application/json, got %q", got)
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response json: %v", err)
			}

			if got := resp["status"]; got != "success" {
				t.Fatalf("expected status success, got %v", got)
			}

			pathParts := strings.Split(strings.TrimPrefix(tt.path, "/api/trace/synthetic/"), "/")
			expectedID := pathParts[0]
			if got := resp["synthetic_id"]; got != expectedID {
				t.Fatalf("expected synthetic_id %q, got %v", expectedID, got)
			}

			latencyVal, ok := resp["latency_ms"].(float64)
			if !ok {
				t.Fatalf("expected numeric latency_ms, got %T", resp["latency_ms"])
			}
			if latencyVal < 5 || latencyVal > 50 {
				t.Fatalf("expected latency_ms between 5 and 50, got %v", latencyVal)
			}
		})
	}
}
