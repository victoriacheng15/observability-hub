package mqttingestor

import (
	"context"
	"errors"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"observability-hub/internal/telemetry"
)

type fakeMQTTToken struct {
	err error
}

func (t *fakeMQTTToken) Wait() bool                     { return true }
func (t *fakeMQTTToken) WaitTimeout(time.Duration) bool { return true }
func (t *fakeMQTTToken) Done() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}
func (t *fakeMQTTToken) Error() error { return t.err }

type fakeRuntimeClient struct {
	subscriptions map[string]byte
	handler       mqtt.MessageHandler
	connectErr    error
}

func (c *fakeRuntimeClient) Connect() mqtt.Token { return &fakeMQTTToken{err: c.connectErr} }
func (c *fakeRuntimeClient) Disconnect(uint)     {}
func (c *fakeRuntimeClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	c.subscriptions = filters
	c.handler = callback
	return &fakeMQTTToken{}
}

func TestRuntimeConfigDefaults(t *testing.T) {
	telemetry.SilenceLogs()

	runtime, err := NewRuntime(RuntimeConfig{})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}

	if runtime.config.BrokerURL != DefaultBrokerURL {
		t.Fatalf("expected default broker %q, got %q", DefaultBrokerURL, runtime.config.BrokerURL)
	}
	if runtime.config.ClientID != DefaultClientID {
		t.Fatalf("expected default client ID %q, got %q", DefaultClientID, runtime.config.ClientID)
	}
	if runtime.config.StaleAfter != DefaultStaleAfter {
		t.Fatalf("expected default stale threshold %s, got %s", DefaultStaleAfter, runtime.config.StaleAfter)
	}
	if len(runtime.config.Topics) != 2 {
		t.Fatalf("expected two default topics, got %d", len(runtime.config.Topics))
	}
}

func TestRuntimeEvaluateMessage(t *testing.T) {
	telemetry.SilenceLogs()

	now := time.Date(2026, 4, 23, 12, 0, 0, 0, time.UTC)
	runtime, err := NewRuntime(RuntimeConfig{StaleAfter: 2 * time.Minute})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}

	first := runtime.evaluateMessage("devices/device-1/telemetry", []byte(payloadJSON("device-1", 1, "running", "power_on", 100, now.Add(-30*time.Second))), now)
	second := runtime.evaluateMessage("devices/device-1/telemetry", []byte(payloadJSON("device-1", 4, "degraded", "power_on", 101, now.Add(-10*time.Second))), now)
	invalid := runtime.evaluateMessage("devices/device-1/telemetry", []byte(`{"device_id":`), now)

	if first.err != nil {
		t.Fatalf("expected first message to succeed, got %v", first.err)
	}
	if first.events[0].name != "mqtt_message_received" {
		t.Fatalf("expected mqtt_message_received, got %q", first.events[0].name)
	}
	if second.err != nil {
		t.Fatalf("expected second message to succeed, got %v", second.err)
	}
	names := eventNames(second.events)
	assertContainsEvent(t, names, "mqtt_message_received")
	assertContainsEvent(t, names, "mqtt_sequence_gap_detected")
	assertContainsEvent(t, names, "device_state_changed")

	if !invalid.invalid {
		t.Fatal("expected invalid payload classification")
	}
	if !errors.Is(invalid.err, ErrMalformedPayload) {
		t.Fatalf("expected malformed payload error, got %v", invalid.err)
	}
	assertContainsEvent(t, eventNames(invalid.events), "mqtt_payload_invalid")
}

func TestRuntimeRunSubscribesToBothTopics(t *testing.T) {
	telemetry.SilenceLogs()

	runtime, err := NewRuntime(RuntimeConfig{})
	if err != nil {
		t.Fatalf("NewRuntime failed: %v", err)
	}

	fakeClient := &fakeRuntimeClient{}
	runtime.newClient = func(*mqtt.ClientOptions) client { return fakeClient }

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := runtime.Run(ctx); err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if len(fakeClient.subscriptions) != len(DefaultTopics) {
		t.Fatalf("expected %d subscriptions, got %d", len(DefaultTopics), len(fakeClient.subscriptions))
	}
	for _, topic := range DefaultTopics {
		if _, ok := fakeClient.subscriptions[topic]; !ok {
			t.Fatalf("expected subscription to %q", topic)
		}
	}
}

func eventNames(events []messageEvent) []string {
	names := make([]string, 0, len(events))
	for _, event := range events {
		names = append(names, event.name)
	}
	return names
}

func assertContainsEvent(t *testing.T, names []string, want string) {
	t.Helper()
	for _, name := range names {
		if name == want {
			return
		}
	}
	t.Fatalf("expected event %q in %v", want, names)
}
