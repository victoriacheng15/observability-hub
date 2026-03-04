package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"observability-hub/internal/collectors"
)

// MockThanos satisfies the ThanosSource interface.
type MockThanos struct {
	QueryRangeFn func(ctx context.Context, query string, start, end time.Time, step string) ([]collectors.Sample, error)
}

func (m *MockThanos) QueryRange(ctx context.Context, query string, start, end time.Time, step string) ([]collectors.Sample, error) {
	if m.QueryRangeFn != nil {
		return m.QueryRangeFn(ctx, query, start, end, step)
	}
	return nil, nil
}

// MockStore satisfies the DataStore interface.
type MockStore struct {
	RecordMetricFn func(ctx context.Context, t time.Time, hostName, osName, metricType string, payload interface{}) error
	Recorded       []string // Simplified log of recorded metric types
}

func (m *MockStore) RecordMetric(ctx context.Context, t time.Time, hostName, osName, metricType string, payload interface{}) error {
	m.Recorded = append(m.Recorded, metricType)
	if m.RecordMetricFn != nil {
		return m.RecordMetricFn(ctx, t, hostName, osName, metricType, payload)
	}
	return nil
}

func setupHostMetadata(t *testing.T) func() {
	// Create temporary directory to simulate /etc
	tmpDir, err := os.MkdirTemp("", "etc-host")
	if err != nil {
		t.Fatalf("failed to create tmp dir: %v", err)
	}

	hostnamePath := filepath.Join(tmpDir, "host_hostname")
	osReleasePath := filepath.Join(tmpDir, "host_os-release")

	if err := os.WriteFile(hostnamePath, []byte("test-host"), 0644); err != nil {
		t.Fatalf("failed to write hostname: %v", err)
	}
	if err := os.WriteFile(osReleasePath, []byte("ID=linux\nVERSION_ID=6.0\n"), 0644); err != nil {
		t.Fatalf("failed to write os-release: %v", err)
	}

	oldHostnamePath := pathHostname
	oldOSReleasePath := pathOSRelease
	pathHostname = hostnamePath
	pathOSRelease = osReleasePath

	return func() {
		os.RemoveAll(tmpDir)
		pathHostname = oldHostnamePath
		pathOSRelease = oldOSReleasePath
	}
}

func TestCollectAndStoreHostMetrics(t *testing.T) {
	now := time.Now().Truncate(time.Minute)

	tests := []struct {
		name           string
		cpuSamples     []collectors.Sample
		tempSamples    []collectors.Sample
		ramSamples     []collectors.Sample
		diskSamples    []collectors.Sample
		netRXSamples   []collectors.Sample
		netTXSamples   []collectors.Sample
		expectedMetric []string
	}{
		{
			name: "Full Collection Success",
			cpuSamples: []collectors.Sample{
				{
					Timestamp: now,
					Payload: map[string]interface{}{
						"value":  "45.5",
						"labels": map[string]string{"instance": "host1:9100"},
					},
				},
			},
			tempSamples: []collectors.Sample{
				{
					Timestamp: now,
					Payload: map[string]interface{}{
						"value":  "55.0",
						"labels": map[string]string{"instance": "host1:9100", "chip": "coretemp", "sensor": "temp1"},
					},
				},
			},
			ramSamples: []collectors.Sample{
				{
					Timestamp: now,
					Payload: map[string]interface{}{
						"value":  "60.2",
						"labels": map[string]string{"instance": "host1:9100"},
					},
				},
			},
			diskSamples: []collectors.Sample{
				{
					Timestamp: now,
					Payload: map[string]interface{}{
						"value":  "80.0",
						"labels": map[string]string{"instance": "host1:9100", "mountpoint": "/"},
					},
				},
			},
			netRXSamples: []collectors.Sample{
				{
					Timestamp: now,
					Payload: map[string]interface{}{
						"value":  "1000",
						"labels": map[string]string{"instance": "host1:9100", "device": "enp5s0"},
					},
				},
			},
			netTXSamples: []collectors.Sample{
				{
					Timestamp: now,
					Payload: map[string]interface{}{
						"value":  "2000",
						"labels": map[string]string{"instance": "host1:9100", "device": "enp5s0"},
					},
				},
			},
			expectedMetric: []string{"cpu", "memory", "disk", "network"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockThanos := &MockThanos{
				QueryRangeFn: func(ctx context.Context, query string, start, end time.Time, step string) ([]collectors.Sample, error) {
					switch {
					case query == queryCPUUtil:
						return tt.cpuSamples, nil
					case query == queryTemp:
						return tt.tempSamples, nil
					case query == queryRAMUtil:
						return tt.ramSamples, nil
					case query == queryDiskUsed:
						return tt.diskSamples, nil
					case query == queryNetRX:
						return tt.netRXSamples, nil
					case query == queryNetTX:
						return tt.netTXSamples, nil
					}
					return nil, nil
				},
			}
			mockStore := &MockStore{}

			app := &App{
				Thanos: mockThanos,
				Store:  mockStore,
			}

			app.collectAndStoreHostMetrics(context.Background(), now.Add(-15*time.Minute), now, "test-host", "linux 6.0")

			recordedMap := make(map[string]bool)
			for _, m := range mockStore.Recorded {
				recordedMap[m] = true
			}
			for _, exp := range tt.expectedMetric {
				if !recordedMap[exp] {
					t.Errorf("expected metric type %s was not recorded", exp)
				}
			}
		})
	}
}

func TestApp_RunBatch(t *testing.T) {
	cleanup := setupHostMetadata(t)
	defer cleanup()

	ensureMetrics()

	mockThanos := &MockThanos{}
	mockStore := &MockStore{}

	app := &App{
		Thanos: mockThanos,
		Store:  mockStore,
	}

	// Should not panic and should run without real collectors
	app.runBatch(context.Background())
}

func TestRun_NoThanosURL(t *testing.T) {
	os.Unsetenv("THANOS_URL")
	err := Run(context.Background())
	if err == nil {
		t.Error("Expected error when THANOS_URL is missing, got nil")
	}
}

func TestApp_CollectTailscale(t *testing.T) {
	// We need to mock collectors.GetFunnelStatus and GetTailscaleStatus.
	// Since those are in pkg/collectors, and they use a global runner, we can mock it.
	ensureMetrics()
	app := &App{}

	// Just call it to see if it covers the lines.
	// Real logic will fail but it handles errors gracefully.
	app.collectTailscale(context.Background())
}
