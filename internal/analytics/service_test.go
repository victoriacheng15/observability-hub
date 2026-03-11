package analytics

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"observability-hub/internal/telemetry"
)

// MockThanos satisfies the ThanosSource interface.
type MockThanos struct {
	QueryRangeFn func(ctx context.Context, query string, start, end time.Time, step string) ([]Sample, error)
}

func (m *MockThanos) QueryRange(ctx context.Context, query string, start, end time.Time, step string) ([]Sample, error) {
	if m.QueryRangeFn != nil {
		return m.QueryRangeFn(ctx, query, start, end, step)
	}
	return nil, nil
}

// MockStore satisfies the DataStore interface.
type MockStore struct {
	RecordMetricFn          func(ctx context.Context, t time.Time, hostName, osName, metricType string, payload interface{}) error
	RecordAnalyticsMetricFn func(ctx context.Context, t time.Time, featureID string, kind MetricKind, value float64, unit string, metadata map[string]interface{}) error
	Recorded                []string // Simplified log of recorded metric types
	AnalyticsRecorded       []string // Simplified log of recorded analytics types
	FeatureIDs              []string // Log of recorded feature IDs
}

func (m *MockStore) RecordMetric(ctx context.Context, t time.Time, hostName, osName, metricType string, payload interface{}) error {
	m.Recorded = append(m.Recorded, metricType)
	if m.RecordMetricFn != nil {
		return m.RecordMetricFn(ctx, t, hostName, osName, metricType, payload)
	}
	return nil
}

func (m *MockStore) RecordAnalyticsMetric(ctx context.Context, t time.Time, featureID string, kind MetricKind, value float64, unit string, metadata map[string]interface{}) error {
	m.AnalyticsRecorded = append(m.AnalyticsRecorded, string(kind))
	m.FeatureIDs = append(m.FeatureIDs, featureID)
	if m.RecordAnalyticsMetricFn != nil {
		return m.RecordAnalyticsMetricFn(ctx, t, featureID, kind, value, unit, metadata)
	}
	return nil
}

// MockResources satisfies the ResourceProvider interface.
type MockResources struct {
	GetEnergyJoulesFn    func(ctx context.Context, start, end time.Time) (float64, error)
	GetContainerEnergyFn func(ctx context.Context, start, end time.Time) (map[string]float64, error)
	GetHostServiceCPUFn  func(ctx context.Context, start, end time.Time) (map[string]float64, error)
}

func (m *MockResources) GetEnergyJoules(ctx context.Context, start, end time.Time) (float64, error) {
	if m.GetEnergyJoulesFn != nil {
		return m.GetEnergyJoulesFn(ctx, start, end)
	}
	return 100.0, nil
}

func (m *MockResources) GetContainerEnergy(ctx context.Context, start, end time.Time) (map[string]float64, error) {
	if m.GetContainerEnergyFn != nil {
		return m.GetContainerEnergyFn(ctx, start, end)
	}
	return map[string]float64{
		"postgres": 20.0,
	}, nil
}

func (m *MockResources) GetHostServiceCPU(ctx context.Context, start, end time.Time) (map[string]float64, error) {
	if m.GetHostServiceCPUFn != nil {
		return m.GetHostServiceCPUFn(ctx, start, end)
	}
	return map[string]float64{
		"ingestion.service": 0.5,
		"proxy.service":     0.5,
	}, nil
}

func (m *MockResources) GetCarbonIntensity(ctx context.Context) (float64, error) {
	return 150.0, nil
}

func (m *MockResources) GetCostFactor(ctx context.Context) (float64, error) {
	return 0.15 / 3600000.0, nil
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
		cpuSamples     []Sample
		tempSamples    []Sample
		ramSamples     []Sample
		diskSamples    []Sample
		netRXSamples   []Sample
		netTXSamples   []Sample
		expectedMetric []string
	}{
		{
			name: "Full Collection Success",
			cpuSamples: []Sample{
				{
					Timestamp: now,
					Payload: map[string]interface{}{
						"value":  "45.5",
						"labels": map[string]string{"instance": "host1:9100"},
					},
				},
			},
			tempSamples: []Sample{
				{
					Timestamp: now,
					Payload: map[string]interface{}{
						"value":  "55.0",
						"labels": map[string]string{"instance": "host1:9100", "chip": "coretemp", "sensor": "temp1"},
					},
				},
			},
			ramSamples: []Sample{
				{
					Timestamp: now,
					Payload: map[string]interface{}{
						"value":  "60.2",
						"labels": map[string]string{"instance": "host1:9100"},
					},
				},
			},
			diskSamples: []Sample{
				{
					Timestamp: now,
					Payload: map[string]interface{}{
						"value":  "80.0",
						"labels": map[string]string{"instance": "host1:9100", "mountpoint": "/"},
					},
				},
			},
			netRXSamples: []Sample{
				{
					Timestamp: now,
					Payload: map[string]interface{}{
						"value":  "1000",
						"labels": map[string]string{"instance": "host1:9100", "device": "enp5s0"},
					},
				},
			},
			netTXSamples: []Sample{
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
				QueryRangeFn: func(ctx context.Context, query string, start, end time.Time, step string) ([]Sample, error) {
					switch {
					case query == QueryCPUUtil:
						return tt.cpuSamples, nil
					case query == QueryTemp:
						return tt.tempSamples, nil
					case query == QueryRAMUtil:
						return tt.ramSamples, nil
					case query == QueryDiskUsed:
						return tt.diskSamples, nil
					case query == QueryNetRX:
						return tt.netRXSamples, nil
					case query == QueryNetTX:
						return tt.netTXSamples, nil
					}
					return nil, nil
				},
			}
			mockStore := &MockStore{}

			s := &Service{
				Thanos: mockThanos,
				Store:  mockStore,
			}

			s.collectAndStoreHostMetrics(context.Background(), now.Add(-15*time.Minute), now, "test-host", "linux 6.0")

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

func TestService_RunBatch(t *testing.T) {
	cleanup := setupHostMetadata(t)
	defer cleanup()

	telemetry.SilenceLogs()
	EnsureMetrics()

	mockThanos := &MockThanos{}
	mockStore := &MockStore{}
	mockResources := &MockResources{}

	s := &Service{
		Thanos:    mockThanos,
		Store:     mockStore,
		Resources: mockResources,
	}

	// Should not panic and should run without real analytics
	s.RunBatch(context.Background())
}

func TestService_ProcessResources(t *testing.T) {
	mockStore := &MockStore{}
	mockResources := &MockResources{}

	s := &Service{
		Store:     mockStore,
		Resources: mockResources,
	}

	s.processResources(context.Background(), time.Now().Add(-15*time.Minute), time.Now(), "test-host", "linux")

	// Total node (3) + pod (3) + 2 attributed services (6) = 12 recordings
	expectedCount := 12
	if len(mockStore.AnalyticsRecorded) != expectedCount {
		t.Errorf("expected %d analytics recordings, got %d", expectedCount, len(mockStore.AnalyticsRecorded))
	}

	featureSet := make(map[string]bool)
	for _, id := range mockStore.FeatureIDs {
		featureSet[id] = true
	}

	// 'postgres' mapped from podMetrics, 'ingestion' and 'proxy' mapped from hostServiceCPU (via .service units)
	expectedFeatures := []string{"node-total", "database-core", "ingestion", "proxy"}
	for _, f := range expectedFeatures {
		if !featureSet[f] {
			t.Errorf("expected feature %s was not recorded", f)
		}
	}
}

func TestService_CollectTailscale(t *testing.T) {
	// Just call it to see if it covers the lines.
	// Real logic will fail but it handles errors gracefully.
	s := &Service{}
	s.collectTailscale(context.Background())
}
