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
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid localhost URL",
			url:     "http://localhost:30090",
			wantErr: false,
		},
		{
			name:    "valid https URL",
			url:     "https://thanos.example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewTelemetryProvider(tt.url)
			if provider.thanosURL != tt.url {
				t.Errorf("expected URL %q, got %q", tt.url, provider.thanosURL)
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
			query: string(make([]byte, 10001)),
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

			provider := NewTelemetryProvider(server.URL)
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

	provider := NewTelemetryProvider(server.URL)
	provider.httpClient.Timeout = 100 * time.Millisecond
	ctx := context.Background()

	_, err := provider.QueryMetrics(ctx, "up")
	if err == nil {
		t.Error("expected timeout error")
	}
}

func TestTelemetryProvider_Close(t *testing.T) {
	provider := NewTelemetryProvider("http://localhost:30090")
	err := provider.Close()
	if err != nil {
		t.Errorf("unexpected error on close: %v", err)
	}
}
