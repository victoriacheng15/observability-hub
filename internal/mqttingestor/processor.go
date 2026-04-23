package mqttingestor

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

const DefaultSchemaVersion = "1"

var (
	ErrMalformedPayload = errors.New("malformed payload")
	ErrInvalidPayload   = errors.New("invalid payload")
)

type Config struct {
	ExpectedSchemaVersion string
	StaleAfter            time.Duration
}

type TelemetryMessage struct {
	SensorID        string  `json:"sensor_id"`
	SchemaVersion   string  `json:"schema_version"`
	DeviceID        string  `json:"device_id"`
	FirmwareVersion string  `json:"firmware_version"`
	DeviceState     string  `json:"device_state"`
	SequenceNumber  uint64  `json:"sequence_number"`
	TelemetryTopic  string  `json:"telemetry_topic"`
	Temperature     float64 `json:"temperature"`
	Voltage         float64 `json:"voltage"`
	Current         float64 `json:"current"`
	PowerUsage      float64 `json:"power_usage"`
	RSSI            float64 `json:"rssi"`
	SNR             float64 `json:"snr"`
	PacketLoss      float64 `json:"packet_loss_percent"`
	FreeHeap        uint64  `json:"free_heap"`
	LoopTimeMS      float64 `json:"loop_time_ms"`
	UptimeSeconds   int64   `json:"uptime_seconds"`
	RebootReason    string  `json:"reboot_reason"`
	Timestamp       string  `json:"timestamp"`
}

type Analysis struct {
	Message             TelemetryMessage
	ReceivedAt          time.Time
	DeviceTime          time.Time
	MessageAge          time.Duration
	IsStale             bool
	IsDuplicate         bool
	IsReboot            bool
	StateChanged        bool
	SequenceGap         uint64
	PreviousFound       bool
	PreviousDeviceState string
}

type deviceState struct {
	SequenceNumber uint64
	UptimeSeconds  int64
	RebootReason   string
	DeviceState    string
}

type Processor struct {
	expectedSchemaVersion string
	staleAfter            time.Duration

	mu      sync.Mutex
	devices map[string]deviceState
}

func NewProcessor(cfg Config) *Processor {
	expectedSchemaVersion := cfg.ExpectedSchemaVersion
	if expectedSchemaVersion == "" {
		expectedSchemaVersion = DefaultSchemaVersion
	}

	return &Processor{
		expectedSchemaVersion: expectedSchemaVersion,
		staleAfter:            cfg.StaleAfter,
		devices:               make(map[string]deviceState),
	}
}

func (p *Processor) Process(payload []byte, receivedAt time.Time) (Analysis, error) {
	msg, deviceTime, err := p.DecodeAndValidate(payload)
	if err != nil {
		return Analysis{}, err
	}

	analysis := Analysis{
		Message:    msg,
		ReceivedAt: receivedAt,
		DeviceTime: deviceTime,
		MessageAge: receivedAt.Sub(deviceTime),
	}

	if p.staleAfter > 0 && analysis.MessageAge > p.staleAfter {
		analysis.IsStale = true
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	previous, ok := p.devices[msg.DeviceID]
	analysis.PreviousFound = ok

	if ok {
		analysis.PreviousDeviceState = previous.DeviceState
		analysis.StateChanged = previous.DeviceState != "" && msg.DeviceState != previous.DeviceState

		switch {
		case msg.SequenceNumber == previous.SequenceNumber:
			analysis.IsDuplicate = true
		case msg.SequenceNumber > previous.SequenceNumber+1:
			analysis.SequenceGap = msg.SequenceNumber - previous.SequenceNumber - 1
		case msg.SequenceNumber < previous.SequenceNumber && rebootEvidence(msg, previous):
			analysis.IsReboot = true
		case msg.SequenceNumber < previous.SequenceNumber:
			analysis.IsDuplicate = true
		}
	}

	p.devices[msg.DeviceID] = deviceState{
		SequenceNumber: msg.SequenceNumber,
		UptimeSeconds:  msg.UptimeSeconds,
		RebootReason:   msg.RebootReason,
		DeviceState:    msg.DeviceState,
	}

	return analysis, nil
}

func (p *Processor) DecodeAndValidate(payload []byte) (TelemetryMessage, time.Time, error) {
	var msg TelemetryMessage
	if err := json.Unmarshal(payload, &msg); err != nil {
		return TelemetryMessage{}, time.Time{}, fmt.Errorf("%w: %v", ErrMalformedPayload, err)
	}

	if err := p.validate(msg); err != nil {
		return TelemetryMessage{}, time.Time{}, err
	}

	deviceTime, err := time.Parse(time.RFC3339, msg.Timestamp)
	if err != nil {
		return TelemetryMessage{}, time.Time{}, fmt.Errorf("%w: invalid timestamp: %v", ErrInvalidPayload, err)
	}

	return msg, deviceTime, nil
}

func (p *Processor) validate(msg TelemetryMessage) error {
	switch {
	case msg.SchemaVersion == "":
		return fmt.Errorf("%w: missing schema_version", ErrInvalidPayload)
	case msg.SchemaVersion != p.expectedSchemaVersion:
		return fmt.Errorf("%w: unexpected schema_version %q", ErrInvalidPayload, msg.SchemaVersion)
	case msg.DeviceID == "":
		return fmt.Errorf("%w: missing device_id", ErrInvalidPayload)
	case msg.DeviceState == "":
		return fmt.Errorf("%w: missing device_state", ErrInvalidPayload)
	case msg.SequenceNumber == 0:
		return fmt.Errorf("%w: missing sequence_number", ErrInvalidPayload)
	case msg.Timestamp == "":
		return fmt.Errorf("%w: missing timestamp", ErrInvalidPayload)
	default:
		return nil
	}
}

func rebootEvidence(msg TelemetryMessage, previous deviceState) bool {
	return msg.UptimeSeconds < previous.UptimeSeconds || msg.RebootReason != previous.RebootReason
}
