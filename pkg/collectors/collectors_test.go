package collectors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestThanosClient_QueryRange(t *testing.T) {
	// Mock Thanos API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock response from Thanos
		responseJSON := `{
			"status": "success",
			"data": {
				"resultType": "matrix",
				"result": [
					{
						"metric": { "instance": "host1", "__name__": "node_cpu_seconds_total" },
						"values": [
							[ 1708531200, "0.5" ],
							[ 1708531260, "0.6" ]
						]
					}
				]
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, responseJSON)
	}))
	defer ts.Close()

	client := NewThanosClient(ts.URL)
	ctx := context.Background()
	start := time.Unix(1708531200, 0)
	end := time.Unix(1708531300, 0)

	samples, err := client.QueryRange(ctx, "node_cpu_seconds_total", start, end, "1m")
	if err != nil {
		t.Fatalf("QueryRange failed: %v", err)
	}

	if len(samples) != 2 {
		t.Errorf("expected 2 samples, got %d", len(samples))
	}

	// Verify first sample
	if samples[0].Timestamp.Unix() != 1708531200 {
		t.Errorf("expected timestamp 1708531200, got %d", samples[0].Timestamp.Unix())
	}
	if samples[0].Payload["value"] != "0.5" {
		t.Errorf("expected value '0.5', got %v", samples[0].Payload["value"])
	}
}

func TestGetFunnelStatus_Parsing(t *testing.T) {
	// Sample output from manual verification
	mockOutput := `
# Funnel on:
#     - https://server.tailc8e03f.ts.net:8443

https://server.tailc8e03f.ts.net:8443 (Funnel on)
|-- / proxy http://127.0.0.1:8085
`

	// This is just to test the logic if we were to extract the parser.
	// For now, we'll verify the structure handles the "Funnel on" substring.
	active := strings.Contains(mockOutput, "(Funnel on)")
	if !active {
		t.Errorf("Expected 'Funnel on' to be detected")
	}

	lines := strings.Split(strings.TrimSpace(mockOutput), "\n")
	var target string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "https://") {
			target = strings.Fields(trimmed)[0]
			break
		}
	}
	if target != "https://server.tailc8e03f.ts.net:8443" {
		t.Errorf("Expected target https://server.tailc8e03f.ts.net:8443, got %s", target)
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
			name:           "API Error Status",
			serverResponse: `{"status": "error", "error": "invalid query"}`,
			serverStatus:   http.StatusOK,
			wantErr:        true,
		},
		{
			name:           "HTTP Error",
			serverResponse: `Internal Server Error`,
			serverStatus:   http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name:           "Malformed JSON",
			serverResponse: `{"status": "success", "data": { "result": [ { "values": [[123, 456]] } ] } }`, // values[1] should be string
			serverStatus:   http.StatusOK,
			wantErr:        false, // Currently the code skips malformed samples rather than failing the whole batch
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

func TestGetTailscaleStatus_Parsing(t *testing.T) {
	mockJSON := `{
		"BackendState": "Running",
		"Self": {
			"HostName": "test-host",
			"TailscaleIPs": ["100.64.0.1"]
		}
	}`

	var status map[string]interface{}
	err := json.Unmarshal([]byte(mockJSON), &status)
	if err != nil {
		t.Fatalf("Failed to unmarshal mock JSON: %v", err)
	}

	if status["BackendState"] != "Running" {
		t.Errorf("Expected BackendState Running, got %v", status["BackendState"])
	}

	self := status["Self"].(map[string]interface{})
	if self["HostName"] != "test-host" {
		t.Errorf("Expected HostName test-host, got %v", self["HostName"])
	}
}

func TestTailscaleStatus_Skipped(t *testing.T) {
	t.Log("Skipping tailscale status real execution test (requires tailscale CLI)")
}
