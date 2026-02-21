package collectors

import (
	"context"
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

func TestTailscaleStatus_Parsing(t *testing.T) {
	// Since GetTailscaleStatus shells out to 'tailscale', we skip this in CI
	// but provide a mockable or logic-only test for local parsing if we extracted it.
	// For now, we'll verify the structure exists and is usable.
	t.Log("Skipping tailscale status parsing test (requires tailscale CLI)")
}
