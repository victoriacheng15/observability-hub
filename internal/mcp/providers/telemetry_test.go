package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestTelemetryProvider_NewAndInitialization(t *testing.T) {
	tests := []struct {
		name      string
		thanosURL string
		lokiURL   string
	}{
		{
			name:      "valid localhost URLs",
			thanosURL: "http://localhost:30090",
			lokiURL:   "http://localhost:30100",
		},
		{
			name:      "valid https URLs",
			thanosURL: "https://thanos.example.com",
			lokiURL:   "https://loki.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewTelemetryProvider(tt.thanosURL, tt.lokiURL)
			if provider.thanosURL != tt.thanosURL {
				t.Errorf("expected thanosURL %q, got %q", tt.thanosURL, provider.thanosURL)
			}
			if provider.lokiURL != tt.lokiURL {
				t.Errorf("expected lokiURL %q, got %q", tt.lokiURL, provider.lokiURL)
			}
			if provider.httpClient == nil {
				t.Error("expected httpClient to be initialized")
			}
		})
	}
}

func TestTelemetryProvider_QueryMetrics(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		setupServer func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errMsg      string
	}{
		{
			name:  "successful query execution",
			query: "up",
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/query" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"success","data":{"resultType":"instant","result":[]}}`))
			},
			wantErr: false,
		},
		{
			name:  "empty query",
			query: "",
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr: true,
			errMsg:  "query cannot be empty",
		},
		{
			name:  "query too long",
			query: string(make([]byte, 5001)),
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr: true,
			errMsg:  "query too long",
		},
		{
			name:  "server returns 500",
			query: "up",
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("Internal Server Error"))
			},
			wantErr: true,
			errMsg:  "status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.setupServer))
			defer server.Close()

			provider := NewTelemetryProvider(server.URL, "http://localhost:30100")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := provider.QueryMetrics(ctx, tt.query)

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == nil {
				t.Error("expected non-nil result for successful query")
			}
		})
	}
}

func TestTelemetryProvider_RequestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewTelemetryProvider(server.URL, "http://localhost:30100")
	provider.httpClient.Timeout = 100 * time.Millisecond
	ctx := context.Background()

	_, err := provider.QueryMetrics(ctx, "up")
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestTelemetryProvider_Close(t *testing.T) {
	provider := NewTelemetryProvider("http://localhost:30090", "http://localhost:30100")
	err := provider.Close()
	if err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}

func TestTelemetryProvider_QueryLogs(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		limit       int
		hours       int
		setupServer func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errMsg      string
	}{
		{
			name:  "successful query execution",
			query: `{job="prometheus"}`,
			limit: 50,
			hours: 1,
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/loki/api/v1/query_range" {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
			},
			wantErr: false,
		},
		{
			name:  "default limit applied when zero",
			query: `{service="mcp.telemetry"}`,
			limit: 0,
			hours: 1,
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("limit") != "100" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
			},
			wantErr: false,
		},
		{
			name:  "custom hours lookback",
			query: `{service="ingestion"}`,
			limit: 100,
			hours: 24,
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status":"success","data":{"resultType":"streams","result":[]}}`))
			},
			wantErr: false,
		},
		{
			name:  "empty query",
			query: "",
			limit: 100,
			hours: 1,
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			wantErr: true,
			errMsg:  "query cannot be empty",
		},
		{
			name:  "loki returns 500",
			query: `{job="prometheus"}`,
			limit: 100,
			hours: 1,
			setupServer: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr: true,
			errMsg:  "status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tt.setupServer))
			defer server.Close()

			provider := NewTelemetryProvider("http://localhost:30090", server.URL)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := provider.QueryLogs(ctx, tt.query, tt.limit, tt.hours)

			if (err != nil) != tt.wantErr {
				t.Errorf("got error %v, want error %v", err, tt.wantErr)
			}
			if !tt.wantErr && result == nil {
				t.Error("expected non-nil result for successful query")
			}
		})
	}
}
