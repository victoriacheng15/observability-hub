package mqttingestor

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestProcessorProcess(t *testing.T) {
	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		processor *Processor
		payloads  []string
		received  []time.Time
		check     func(t *testing.T, analyses []Analysis, err error)
	}{
		{
			name:      "valid payload computes message age",
			processor: NewProcessor(Config{StaleAfter: 5 * time.Minute}),
			payloads: []string{
				payloadJSON("device-1", 1, "running", "power_on", 120, now.Add(-30*time.Second)),
			},
			received: []time.Time{now},
			check: func(t *testing.T, analyses []Analysis, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if len(analyses) != 1 {
					t.Fatalf("expected one analysis, got %d", len(analyses))
				}
				got := analyses[0]
				if got.Message.DeviceID != "device-1" {
					t.Fatalf("expected device_id device-1, got %q", got.Message.DeviceID)
				}
				if got.MessageAge != 30*time.Second {
					t.Fatalf("expected message age 30s, got %s", got.MessageAge)
				}
				if got.IsStale {
					t.Fatal("expected message not to be stale")
				}
			},
		},
		{
			name:      "malformed payload returns malformed error",
			processor: NewProcessor(Config{}),
			payloads:  []string{`{"device_id":`},
			received:  []time.Time{now},
			check: func(t *testing.T, analyses []Analysis, err error) {
				t.Helper()
				if err == nil {
					t.Fatal("expected malformed payload error")
				}
				if !errors.Is(err, ErrMalformedPayload) {
					t.Fatalf("expected ErrMalformedPayload, got %v", err)
				}
				if len(analyses) != 0 {
					t.Fatalf("expected no analyses, got %d", len(analyses))
				}
			},
		},
		{
			name:      "invalid schema returns invalid error",
			processor: NewProcessor(Config{}),
			payloads: []string{
				payloadWithSchema("device-1", 1, "2", "running", "power_on", 100, now),
			},
			received: []time.Time{now},
			check: func(t *testing.T, analyses []Analysis, err error) {
				t.Helper()
				if err == nil {
					t.Fatal("expected invalid payload error")
				}
				if !errors.Is(err, ErrInvalidPayload) {
					t.Fatalf("expected ErrInvalidPayload, got %v", err)
				}
				if !strings.Contains(err.Error(), "schema_version") {
					t.Fatalf("expected schema_version error, got %v", err)
				}
			},
		},
		{
			name:      "sequence gap is detected",
			processor: NewProcessor(Config{}),
			payloads: []string{
				payloadJSON("device-1", 1, "running", "power_on", 120, now.Add(-2*time.Second)),
				payloadJSON("device-1", 4, "running", "power_on", 123, now.Add(-1*time.Second)),
			},
			received: []time.Time{now, now},
			check: func(t *testing.T, analyses []Analysis, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if analyses[1].SequenceGap != 2 {
					t.Fatalf("expected sequence gap 2, got %d", analyses[1].SequenceGap)
				}
				if analyses[1].IsDuplicate {
					t.Fatal("did not expect duplicate flag")
				}
			},
		},
		{
			name:      "duplicate sequence is detected",
			processor: NewProcessor(Config{}),
			payloads: []string{
				payloadJSON("device-1", 5, "running", "power_on", 120, now.Add(-2*time.Second)),
				payloadJSON("device-1", 5, "running", "power_on", 121, now.Add(-1*time.Second)),
			},
			received: []time.Time{now, now},
			check: func(t *testing.T, analyses []Analysis, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !analyses[1].IsDuplicate {
					t.Fatal("expected duplicate flag")
				}
				if analyses[1].IsReboot {
					t.Fatal("did not expect reboot flag")
				}
			},
		},
		{
			name:      "stale timestamp is detected",
			processor: NewProcessor(Config{StaleAfter: 2 * time.Minute}),
			payloads: []string{
				payloadJSON("device-1", 1, "running", "power_on", 120, now.Add(-3*time.Minute)),
			},
			received: []time.Time{now},
			check: func(t *testing.T, analyses []Analysis, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !analyses[0].IsStale {
					t.Fatal("expected stale flag")
				}
			},
		},
		{
			name:      "sequence reset with lower uptime is treated as reboot",
			processor: NewProcessor(Config{}),
			payloads: []string{
				payloadJSON("device-1", 9, "running", "power_on", 900, now.Add(-4*time.Second)),
				payloadJSON("device-1", 1, "rebooting", "brownout", 2, now.Add(-1*time.Second)),
			},
			received: []time.Time{now, now},
			check: func(t *testing.T, analyses []Analysis, err error) {
				t.Helper()
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if !analyses[1].IsReboot {
					t.Fatal("expected reboot flag")
				}
				if analyses[1].IsDuplicate {
					t.Fatal("did not expect duplicate flag")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				analyses []Analysis
				err      error
			)

			for i, payload := range tt.payloads {
				var analysis Analysis
				analysis, err = tt.processor.Process([]byte(payload), tt.received[i])
				if err != nil {
					break
				}
				analyses = append(analyses, analysis)
			}

			tt.check(t, analyses, err)
		})
	}
}

func payloadJSON(deviceID string, sequence uint64, state, rebootReason string, uptime int64, ts time.Time) string {
	return payloadWithSchema(deviceID, sequence, DefaultSchemaVersion, state, rebootReason, uptime, ts)
}

func payloadWithSchema(deviceID string, sequence uint64, schemaVersion, state, rebootReason string, uptime int64, ts time.Time) string {
	return fmt.Sprintf(`{
		"sensor_id":"sensor-1",
		"schema_version":"%s",
		"device_id":"%s",
		"firmware_version":"learning-lab-0.1.0",
		"device_state":"%s",
		"sequence_number":%d,
		"telemetry_topic":"sensors/thermal",
		"temperature":23.7,
		"voltage":3.71,
		"current":0.12,
		"power_usage":0.45,
		"rssi":-66,
		"snr":18.4,
		"packet_loss_percent":1.2,
		"free_heap":184320,
		"loop_time_ms":8.5,
		"uptime_seconds":%d,
		"reboot_reason":"%s",
		"timestamp":"%s"
	}`, schemaVersion, deviceID, state, sequence, uptime, rebootReason, ts.Format(time.RFC3339))
}
