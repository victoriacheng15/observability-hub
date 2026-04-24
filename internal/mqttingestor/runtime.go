package mqttingestor

import (
	"context"
	"errors"
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"observability-hub/internal/telemetry"
)

const (
	DefaultBrokerURL         = "tcp://emqx.observability:1883"
	DefaultClientID          = "mqtt-ingestor"
	DefaultStaleAfter        = 30 * time.Second
	defaultMetricDescription = "Synthetic hardware telemetry signal"
)

var DefaultTopics = []string{
	"devices/+/telemetry",
	"sensors/thermal",
}

type client interface {
	Connect() mqtt.Token
	Disconnect(quiesce uint)
	SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token
}

type RuntimeConfig struct {
	BrokerURL             string
	ClientID              string
	Environment           string
	ExpectedSchemaVersion string
	StaleAfter            time.Duration
	Topics                []string
}

type Runtime struct {
	config    RuntimeConfig
	processor *Processor
	metrics   runtimeMetrics
	newClient func(*mqtt.ClientOptions) client
	now       func() time.Time
}

type runtimeMetrics struct {
	messagesTotal        telemetry.Int64Counter
	invalidMessagesTotal telemetry.Int64Counter
	sequenceGapTotal     telemetry.Int64Counter
	duplicatesTotal      telemetry.Int64Counter
	messageAge           telemetry.Int64Histogram
	rebootTotal          telemetry.Int64Counter
	temperature          telemetry.Float64Histogram
	voltage              telemetry.Float64Histogram
	current              telemetry.Float64Histogram
	power                telemetry.Float64Histogram
	rssi                 telemetry.Float64Histogram
	snr                  telemetry.Float64Histogram
	packetLoss           telemetry.Float64Histogram
	freeHeap             telemetry.Int64Histogram
	loopTime             telemetry.Float64Histogram
	uptime               telemetry.Int64Histogram
}

type messageEvent struct {
	name  string
	attrs []any
}

type messageResult struct {
	analysis Analysis
	invalid  bool
	err      error
	events   []messageEvent
}

func NewRuntime(cfg RuntimeConfig) (*Runtime, error) {
	cfg = cfg.withDefaults()

	metrics, err := newRuntimeMetrics()
	if err != nil {
		return nil, err
	}

	return &Runtime{
		config:    cfg,
		processor: NewProcessor(Config{ExpectedSchemaVersion: cfg.ExpectedSchemaVersion, StaleAfter: cfg.StaleAfter}),
		metrics:   metrics,
		newClient: func(opts *mqtt.ClientOptions) client { return mqtt.NewClient(opts) },
		now:       time.Now,
	}, nil
}

func (r *Runtime) Run(ctx context.Context) error {
	opts := mqtt.NewClientOptions().AddBroker(r.config.BrokerURL)
	opts.SetClientID(r.config.ClientID)
	opts.SetCleanSession(true)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(5 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		r.subscribe(c)
	})
	opts.SetConnectionLostHandler(func(_ mqtt.Client, err error) {
		telemetry.Warn("mqtt_connection_lost", "error", err)
	})

	mqttClient := r.newClient(opts)
	token := mqttClient.Connect()
	if token.Wait() && token.Error() != nil {
		return fmt.Errorf("connect mqtt client: %w", token.Error())
	}
	defer mqttClient.Disconnect(250)

	r.subscribe(mqttClient)

	telemetry.Info("mqtt_ingestor_started", "broker", r.config.BrokerURL, "topics", r.config.Topics)

	<-ctx.Done()
	telemetry.Info("mqtt_ingestor_stopped", "reason", ctx.Err())
	return nil
}

func (r *Runtime) subscribe(c client) {
	token := c.SubscribeMultiple(r.subscriptionFilters(), r.handleMQTTMessage)
	if token.Wait() && token.Error() != nil {
		telemetry.Error("mqtt_subscription_failed", "error", token.Error())
		return
	}
	telemetry.Info("mqtt_subscription_established", "topics", r.config.Topics)
}

func (r *Runtime) evaluateMessage(topic string, payload []byte, receivedAt time.Time) messageResult {
	analysis, err := r.processor.Process(payload, receivedAt)
	if err != nil {
		if errors.Is(err, ErrMalformedPayload) || errors.Is(err, ErrInvalidPayload) {
			return messageResult{
				invalid: true,
				err:     err,
				events: []messageEvent{{
					name: "mqtt_payload_invalid",
					attrs: []any{
						"topic", topic,
						"error", err.Error(),
					},
				}},
			}
		}

		return messageResult{err: err}
	}

	result := messageResult{
		analysis: analysis,
		events: []messageEvent{{
			name: "mqtt_message_received",
			attrs: []any{
				"topic", topic,
				"device_id", analysis.Message.DeviceID,
				"device_state", analysis.Message.DeviceState,
				"sequence_number", analysis.Message.SequenceNumber,
			},
		}},
	}

	if analysis.SequenceGap > 0 {
		result.events = append(result.events, messageEvent{
			name: "mqtt_sequence_gap_detected",
			attrs: []any{
				"topic", topic,
				"device_id", analysis.Message.DeviceID,
				"sequence_gap", analysis.SequenceGap,
				"sequence_number", analysis.Message.SequenceNumber,
			},
		})
	}

	if analysis.IsDuplicate {
		result.events = append(result.events, messageEvent{
			name: "mqtt_duplicate_sequence_detected",
			attrs: []any{
				"topic", topic,
				"device_id", analysis.Message.DeviceID,
				"sequence_number", analysis.Message.SequenceNumber,
			},
		})
	}

	if analysis.IsReboot {
		result.events = append(result.events, messageEvent{
			name: "device_reboot_detected",
			attrs: []any{
				"topic", topic,
				"device_id", analysis.Message.DeviceID,
				"reboot_reason", analysis.Message.RebootReason,
			},
		})
	}

	if analysis.StateChanged {
		result.events = append(result.events, messageEvent{
			name: "device_state_changed",
			attrs: []any{
				"topic", topic,
				"device_id", analysis.Message.DeviceID,
				"previous_state", analysis.PreviousDeviceState,
				"device_state", analysis.Message.DeviceState,
			},
		})
	}

	if analysis.IsStale {
		result.events = append(result.events, messageEvent{
			name: "device_message_stale",
			attrs: []any{
				"topic", topic,
				"device_id", analysis.Message.DeviceID,
				"message_age_ms", analysis.MessageAge.Milliseconds(),
			},
		})
	}

	return result
}

func (r *Runtime) recordResult(ctx context.Context, result messageResult) {
	if result.err != nil && !result.invalid {
		telemetry.Error("mqtt_message_processing_failed", "error", result.err)
		return
	}

	for _, event := range result.events {
		if event.name == "mqtt_payload_invalid" {
			telemetry.Warn(event.name, event.attrs...)
			continue
		}
		telemetry.Info(event.name, event.attrs...)
	}

	if result.invalid {
		telemetry.AddInt64Counter(ctx, r.metrics.invalidMessagesTotal, 1, telemetry.StringAttribute("environment", r.config.Environment))
		return
	}

	analysis := result.analysis
	commonAttrs := []telemetry.Attribute{
		telemetry.StringAttribute("environment", r.config.Environment),
		telemetry.StringAttribute("device_id", analysis.Message.DeviceID),
		telemetry.StringAttribute("firmware_version", analysis.Message.FirmwareVersion),
		telemetry.StringAttribute("device_state", analysis.Message.DeviceState),
		telemetry.StringAttribute("topic", analysis.Message.TelemetryTopic),
	}

	telemetry.AddInt64Counter(ctx, r.metrics.messagesTotal, 1, commonAttrs...)
	telemetry.RecordInt64Histogram(ctx, r.metrics.messageAge, analysis.MessageAge.Milliseconds(), commonAttrs...)
	telemetry.RecordFloat64Histogram(ctx, r.metrics.temperature, analysis.Message.Temperature, commonAttrs...)
	telemetry.RecordFloat64Histogram(ctx, r.metrics.voltage, analysis.Message.Voltage, commonAttrs...)
	telemetry.RecordFloat64Histogram(ctx, r.metrics.current, analysis.Message.Current, commonAttrs...)
	telemetry.RecordFloat64Histogram(ctx, r.metrics.power, analysis.Message.PowerUsage, commonAttrs...)
	telemetry.RecordFloat64Histogram(ctx, r.metrics.rssi, analysis.Message.RSSI, commonAttrs...)
	telemetry.RecordFloat64Histogram(ctx, r.metrics.snr, analysis.Message.SNR, commonAttrs...)
	telemetry.RecordFloat64Histogram(ctx, r.metrics.packetLoss, analysis.Message.PacketLoss, commonAttrs...)
	telemetry.RecordInt64Histogram(ctx, r.metrics.freeHeap, int64(analysis.Message.FreeHeap), commonAttrs...)
	telemetry.RecordFloat64Histogram(ctx, r.metrics.loopTime, analysis.Message.LoopTimeMS, commonAttrs...)
	telemetry.RecordInt64Histogram(ctx, r.metrics.uptime, analysis.Message.UptimeSeconds, append(commonAttrs, telemetry.StringAttribute("reboot_reason", analysis.Message.RebootReason))...)

	if analysis.SequenceGap > 0 {
		telemetry.AddInt64Counter(ctx, r.metrics.sequenceGapTotal, int64(analysis.SequenceGap), commonAttrs...)
	}
	if analysis.IsDuplicate {
		telemetry.AddInt64Counter(ctx, r.metrics.duplicatesTotal, 1, commonAttrs...)
	}
	if analysis.IsReboot {
		telemetry.AddInt64Counter(
			ctx,
			r.metrics.rebootTotal,
			1,
			append(commonAttrs, telemetry.StringAttribute("reboot_reason", analysis.Message.RebootReason))...,
		)
	}
}

func (r *Runtime) handleMQTTMessage(_ mqtt.Client, msg mqtt.Message) {
	result := r.evaluateMessage(msg.Topic(), msg.Payload(), r.now().UTC())
	r.recordResult(context.Background(), result)
}

func (r *Runtime) subscriptionFilters() map[string]byte {
	filters := make(map[string]byte, len(r.config.Topics))
	for _, topic := range r.config.Topics {
		filters[topic] = 1
	}
	return filters
}

func (cfg RuntimeConfig) withDefaults() RuntimeConfig {
	if cfg.BrokerURL == "" {
		cfg.BrokerURL = DefaultBrokerURL
	}
	if cfg.ClientID == "" {
		cfg.ClientID = DefaultClientID
	}
	if cfg.Environment == "" {
		cfg.Environment = "unknown"
	}
	if cfg.StaleAfter <= 0 {
		cfg.StaleAfter = DefaultStaleAfter
	}
	if len(cfg.Topics) == 0 {
		cfg.Topics = append([]string(nil), DefaultTopics...)
	}
	return cfg
}

func newRuntimeMetrics() (runtimeMetrics, error) {
	meter := telemetry.GetMeter("mqtt-ingestor.runtime")

	messagesTotal, err := telemetry.NewInt64Counter(meter, "hardware.telemetry.messages.total", defaultMetricDescription)
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create messages counter: %w", err)
	}
	invalidMessagesTotal, err := telemetry.NewInt64Counter(meter, "hardware.telemetry.invalid_messages.total", defaultMetricDescription)
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create invalid messages counter: %w", err)
	}
	sequenceGapTotal, err := telemetry.NewInt64Counter(meter, "hardware.telemetry.sequence_gap.total", defaultMetricDescription)
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create sequence gap counter: %w", err)
	}
	duplicatesTotal, err := telemetry.NewInt64Counter(meter, "hardware.telemetry.duplicates.total", defaultMetricDescription)
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create duplicates counter: %w", err)
	}
	messageAge, err := telemetry.NewInt64Histogram(meter, "hardware.telemetry.message_age", defaultMetricDescription, "ms")
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create message age histogram: %w", err)
	}
	rebootTotal, err := telemetry.NewInt64Counter(meter, "hardware.reboot.total", defaultMetricDescription)
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create reboot counter: %w", err)
	}
	temperature, err := telemetry.NewFloat64Histogram(meter, "hardware.sensor.temperature", defaultMetricDescription, "C")
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create temperature histogram: %w", err)
	}
	voltage, err := telemetry.NewFloat64Histogram(meter, "hardware.sensor.voltage", defaultMetricDescription, "V")
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create voltage histogram: %w", err)
	}
	current, err := telemetry.NewFloat64Histogram(meter, "hardware.sensor.current", defaultMetricDescription, "A")
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create current histogram: %w", err)
	}
	power, err := telemetry.NewFloat64Histogram(meter, "hardware.sensor.power", defaultMetricDescription, "W")
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create power histogram: %w", err)
	}
	rssi, err := telemetry.NewFloat64Histogram(meter, "hardware.radio.rssi", defaultMetricDescription, "dBm")
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create rssi histogram: %w", err)
	}
	snr, err := telemetry.NewFloat64Histogram(meter, "hardware.radio.snr", defaultMetricDescription, "dB")
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create snr histogram: %w", err)
	}
	packetLoss, err := telemetry.NewFloat64Histogram(meter, "hardware.radio.packet_loss", defaultMetricDescription, "%")
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create packet loss histogram: %w", err)
	}
	freeHeap, err := telemetry.NewInt64Histogram(meter, "hardware.runtime.free_heap", defaultMetricDescription, "By")
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create free heap histogram: %w", err)
	}
	loopTime, err := telemetry.NewFloat64Histogram(meter, "hardware.runtime.loop_time", defaultMetricDescription, "ms")
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create loop time histogram: %w", err)
	}
	uptime, err := telemetry.NewInt64Histogram(meter, "hardware.runtime.uptime", defaultMetricDescription, "s")
	if err != nil {
		return runtimeMetrics{}, fmt.Errorf("create uptime histogram: %w", err)
	}

	return runtimeMetrics{
		messagesTotal:        messagesTotal,
		invalidMessagesTotal: invalidMessagesTotal,
		sequenceGapTotal:     sequenceGapTotal,
		duplicatesTotal:      duplicatesTotal,
		messageAge:           messageAge,
		rebootTotal:          rebootTotal,
		temperature:          temperature,
		voltage:              voltage,
		current:              current,
		power:                power,
		rssi:                 rssi,
		snr:                  snr,
		packetLoss:           packetLoss,
		freeHeap:             freeHeap,
		loopTime:             loopTime,
		uptime:               uptime,
	}, nil
}
