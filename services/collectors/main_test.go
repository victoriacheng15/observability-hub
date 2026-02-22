package main

import (
	"context"
	"testing"
	"time"

	"collectors"
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

func TestCollectAndStoreHostMetrics(t *testing.T) {
	now := time.Now().Truncate(time.Minute)

	tests := []struct {
		name           string
		cpuSamples     []collectors.Sample
		tempSamples    []collectors.Sample
		ramSamples     []collectors.Sample
		expectedMetric []string
	}{
		{
			name: "Successful CPU and Temp Merge",
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
						"labels": map[string]string{"instance": "host1:9100", "label": "Package id 0", "sensor": "temp1"},
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
			expectedMetric: []string{"cpu", "memory"},
		},
		{
			name:           "Empty results",
			cpuSamples:     []collectors.Sample{},
			tempSamples:    []collectors.Sample{},
			ramSamples:     []collectors.Sample{},
			expectedMetric: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockThanos := &MockThanos{
				QueryRangeFn: func(ctx context.Context, query string, start, end time.Time, step string) ([]collectors.Sample, error) {
					switch query {
					case queryCPUUtil:
						return tt.cpuSamples, nil
					case queryTemp:
						return tt.tempSamples, nil
					case queryRAMUtil:
						return tt.ramSamples, nil
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

			if len(mockStore.Recorded) != len(tt.expectedMetric) {
				t.Errorf("expected %d metrics, got %d", len(tt.expectedMetric), len(mockStore.Recorded))
			}

			// Verify specific metric types were recorded
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
