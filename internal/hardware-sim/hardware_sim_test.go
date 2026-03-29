package hardwaresim

import (
	"context"
	"encoding/json"
	"math/rand"
	"regexp"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

type fakeToken struct {
	once sync.Once
	ch   chan struct{}
	err  error
}

func newFakeToken(err error) *fakeToken {
	t := &fakeToken{ch: make(chan struct{}), err: err}
	close(t.ch)
	return t
}

func (t *fakeToken) Wait() bool                     { <-t.ch; return true }
func (t *fakeToken) WaitTimeout(time.Duration) bool { return t.Wait() }
func (t *fakeToken) Done() <-chan struct{}          { return t.ch }
func (t *fakeToken) Error() error                   { return t.err }

type published struct {
	topic    string
	qos      byte
	retained bool
	payload  interface{}
}

type fakeMQTTClient struct {
	mu        sync.Mutex
	published []published
}

func (c *fakeMQTTClient) IsConnected() bool      { return true }
func (c *fakeMQTTClient) IsConnectionOpen() bool { return true }
func (c *fakeMQTTClient) Connect() mqtt.Token    { return newFakeToken(nil) }
func (c *fakeMQTTClient) Disconnect(uint)        {}
func (c *fakeMQTTClient) Subscribe(string, byte, mqtt.MessageHandler) mqtt.Token {
	return newFakeToken(nil)
}
func (c *fakeMQTTClient) SubscribeMultiple(map[string]byte, mqtt.MessageHandler) mqtt.Token {
	return newFakeToken(nil)
}
func (c *fakeMQTTClient) Unsubscribe(...string) mqtt.Token     { return newFakeToken(nil) }
func (c *fakeMQTTClient) AddRoute(string, mqtt.MessageHandler) {}
func (c *fakeMQTTClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.NewOptionsReader(mqtt.NewClientOptions())
}

func (c *fakeMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	c.mu.Lock()
	c.published = append(c.published, published{
		topic:    topic,
		qos:      qos,
		retained: retained,
		payload:  payload,
	})
	c.mu.Unlock()
	return newFakeToken(nil)
}

func (c *fakeMQTTClient) Publishes() []published {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]published, len(c.published))
	copy(out, c.published)
	return out
}

type fakeMessage struct {
	payload []byte
}

func (m *fakeMessage) Duplicate() bool   { return false }
func (m *fakeMessage) Qos() byte         { return 0 }
func (m *fakeMessage) Retained() bool    { return false }
func (m *fakeMessage) Topic() string     { return "" }
func (m *fakeMessage) MessageID() uint16 { return 0 }
func (m *fakeMessage) Payload() []byte   { return m.payload }
func (m *fakeMessage) Ack()              {}

func TestSensor_generateData_SpikeIncreasesPower(t *testing.T) {
	s := &Sensor{ID: "sensor-1"}

	rand.Seed(123)
	base := s.generateData()

	s.mu.Lock()
	s.isSpiking = true
	s.spikeIntensity = "high"
	s.mu.Unlock()

	rand.Seed(123)
	spike := s.generateData()

	// High intensity adds at least 25W, regardless of host temperature override logic.
	if spike.PowerUsage <= base.PowerUsage+20 {
		t.Fatalf("expected spike power to be significantly higher, base=%v spike=%v", base.PowerUsage, spike.PowerUsage)
	}
	if spike.SensorID != "sensor-1" {
		t.Fatalf("expected sensor_id to be set, got %q", spike.SensorID)
	}
	if spike.Timestamp == "" {
		t.Fatal("expected timestamp to be set")
	}
}

func TestSensor_handleChaos_SetsAndClearsSpike(t *testing.T) {
	s := &Sensor{ID: "sensor-1"}

	cmd := ChaosCommand{
		Command:   "spike",
		Duration:  "5ms",
		Intensity: "medium",
	}
	b, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("marshal chaos command: %v", err)
	}

	s.handleChaos(nil, &fakeMessage{payload: b})

	s.mu.Lock()
	spiking := s.isSpiking
	intensity := s.spikeIntensity
	s.mu.Unlock()

	if !spiking || intensity != "medium" {
		t.Fatalf("expected spike to be active immediately, spiking=%v intensity=%q", spiking, intensity)
	}

	time.Sleep(20 * time.Millisecond)

	s.mu.Lock()
	spiking = s.isSpiking
	intensity = s.spikeIntensity
	s.mu.Unlock()

	if spiking || intensity != "" {
		t.Fatalf("expected spike to be cleared, spiking=%v intensity=%q", spiking, intensity)
	}
}

func TestChaosController_injectChaos_NoPods_NoPublish(t *testing.T) {
	ctx := context.Background()
	k8s := fake.NewSimpleClientset()

	mq := &fakeMQTTClient{}
	c := &ChaosController{Namespace: "default"}

	rand.Seed(1)
	c.injectChaos(ctx, k8s, mq)

	if got := len(mq.Publishes()); got != 0 {
		t.Fatalf("expected no publish calls, got %d", got)
	}
}

func TestChaosController_injectChaos_PublishesToSensorTopic(t *testing.T) {
	ctx := context.Background()

	p1 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sensor-a",
			Namespace: "default",
			Labels:    map[string]string{"app": "sensor-fleet"},
		},
	}
	p2 := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sensor-b",
			Namespace: "default",
			Labels:    map[string]string{"app": "sensor-fleet"},
		},
	}
	k8s := fake.NewSimpleClientset(p1, p2)

	mq := &fakeMQTTClient{}
	c := &ChaosController{Namespace: "default"}

	rand.Seed(2)
	c.injectChaos(ctx, k8s, mq)

	pubs := mq.Publishes()
	if len(pubs) != 1 {
		t.Fatalf("expected 1 publish call, got %d", len(pubs))
	}
	pub := pubs[0]

	if pub.qos != 1 {
		t.Fatalf("expected qos=1, got %d", pub.qos)
	}
	if pub.retained {
		t.Fatal("expected retained=false")
	}

	topicRe := regexp.MustCompile(`^sensors/(sensor-a|sensor-b)/chaos$`)
	if !topicRe.MatchString(pub.topic) {
		t.Fatalf("unexpected topic %q", pub.topic)
	}

	payload, ok := pub.payload.(string)
	if !ok {
		t.Fatalf("expected string payload, got %T", pub.payload)
	}
	payloadRe := regexp.MustCompile(`^\{"command": "spike", "duration": "\d+s", "intensity": "(low|medium|high)"\}$`)
	if !payloadRe.MatchString(payload) {
		t.Fatalf("unexpected payload %q", payload)
	}
}

func TestRunLoops(t *testing.T) {
	// These tests use short-lived contexts to exercise the Run loops.
	// Since real MQTT and K8s configuration is hard-coded to look for in-cluster/env,
	// we test the graceful shutdown via context cancellation.

	tests := []struct {
		name string
		run  func(ctx context.Context) error
	}{
		{
			name: "Sensor Run Shutdown",
			run: func(ctx context.Context) error {
				s := &Sensor{ID: "test", MqttBroker: "tcp://localhost:1883"}
				// This will fail connection but should still exit on ctx.Done()
				// We wrap in go routine or use short timeout
				return s.Run(ctx)
			},
		},
		{
			name: "ChaosController Run Shutdown",
			run: func(ctx context.Context) error {
				c := &ChaosController{MqttBroker: "tcp://localhost:1883", Namespace: "default"}
				return c.Run(ctx)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
			defer cancel()

			err := tt.run(ctx)
			// Connection errors are expected because we are not starting a real broker,
			// but we want to see the statements being executed.
			if err != nil && !regexp.MustCompile("failed to connect|failed to get in-cluster config").MatchString(err.Error()) {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
