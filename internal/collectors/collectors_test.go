package collectors

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type MockRunner struct {
	RunFn func(ctx context.Context, name string, arg ...string) ([]byte, error)
}

func (m *MockRunner) Run(ctx context.Context, name string, arg ...string) ([]byte, error) {
	if m.RunFn != nil {
		return m.RunFn(ctx, name, arg...)
	}
	return nil, nil
}

func TestThanosClient_QueryRange(t *testing.T) {
	tests := []struct {
		name         string
		responseJSON string
		status       int
		wantErr      bool
		wantCount    int
	}{
		{
			name: "Success",
			responseJSON: `{
				"status": "success",
				"data": {
					"resultType": "matrix",
					"result": [
						{
							"metric": { "instance": "host1" },
							"values": [
								[ 1708531200, "0.5" ]
							]
						}
					]
				}
			}`,
			status:    http.StatusOK,
			wantErr:   false,
			wantCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				fmt.Fprint(w, tt.responseJSON)
			}))
			defer ts.Close()

			client := NewThanosClient(ts.URL)
			samples, err := client.QueryRange(context.Background(), "test", time.Now(), time.Now(), "1m")
			if (err != nil) != tt.wantErr {
				t.Fatalf("QueryRange() error = %v, wantErr %v", err, tt.wantErr)
			}

			if len(samples) != tt.wantCount {
				t.Errorf("expected %d samples, got %d", tt.wantCount, len(samples))
			}
		})
	}
}

func TestGetFunnelStatus(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantActive bool
		wantTarget string
	}{
		{
			name: "Active Funnel",
			mockOutput: `
https://server.ts.net:8443 (Funnel on)
|-- / proxy http://127.0.0.1:8085
`,
			mockErr:    nil,
			wantActive: true,
			wantTarget: "https://server.ts.net:8443",
		},
		{
			name:       "Inactive Funnel",
			mockOutput: "Funnel is off",
			mockErr:    errors.New("exit status 1"),
			wantActive: false,
			wantTarget: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner = &MockRunner{
				RunFn: func(ctx context.Context, name string, arg ...string) ([]byte, error) {
					return []byte(tt.mockOutput), tt.mockErr
				},
			}
			status, err := GetFunnelStatus(context.Background())
			if err != nil {
				t.Errorf("GetFunnelStatus() error = %v", err)
			}
			if status.Active != tt.wantActive {
				t.Errorf("Active = %v, want %v", status.Active, tt.wantActive)
			}
			if status.Target != tt.wantTarget {
				t.Errorf("Target = %v, want %v", status.Target, tt.wantTarget)
			}
		})
	}
}

func TestGetTailscaleStatus(t *testing.T) {
	oldRunner := runner
	defer func() { runner = oldRunner }()

	tests := []struct {
		name       string
		mockOutput string
		mockErr    error
		wantErr    bool
	}{
		{
			name:       "Success",
			mockOutput: `{"BackendState": "Running"}`,
			mockErr:    nil,
			wantErr:    false,
		},
		{
			name:       "Command Error",
			mockOutput: "",
			mockErr:    errors.New("failed"),
			wantErr:    true,
		},
		{
			name:       "Invalid JSON",
			mockOutput: "invalid",
			mockErr:    nil,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner = &MockRunner{
				RunFn: func(ctx context.Context, name string, arg ...string) ([]byte, error) {
					return []byte(tt.mockOutput), tt.mockErr
				},
			}
			_, err := GetTailscaleStatus(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetTailscaleStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestThanosClient_QueryRange_Errors(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		wantErr        bool
	}{
		{
			name:           "HTTP Error",
			serverResponse: "Internal Server Error",
			serverStatus:   http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name:           "Malformed Response Status",
			serverResponse: `{"status": "error"}`,
			serverStatus:   http.StatusOK,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				fmt.Fprint(w, tt.serverResponse)
			}))
			defer ts.Close()

			client := NewThanosClient(ts.URL)
			_, err := client.QueryRange(context.Background(), "test", time.Now(), time.Now(), "1m")
			if (err != nil) != tt.wantErr {
				t.Errorf("QueryRange() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRealCommandRunner_Run(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
		want    string
		wantErr bool
	}{
		{
			name:    "Echo Success",
			command: "echo",
			args:    []string{"hello"},
			want:    "hello\n",
			wantErr: false,
		},
	}

	r := &RealCommandRunner{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := r.Run(context.Background(), tt.command, tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("RealCommandRunner.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
			if string(out) != tt.want {
				t.Errorf("Expected %q, got %q", tt.want, string(out))
			}
		})
	}
}
