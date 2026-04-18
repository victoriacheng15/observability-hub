package hardwaresim

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	DefaultThermalTelemetryTopic = "sensors/thermal"
	DefaultFirmwareVersion       = "dev"
)

// SensorData represents the synthetic sensor payload.
type SensorData struct {
	SensorID        string  `json:"sensor_id"`
	DeviceID        string  `json:"device_id"`
	FirmwareVersion string  `json:"firmware_version"`
	TelemetryTopic  string  `json:"telemetry_topic"`
	Temperature     float64 `json:"temperature"`
	Voltage         float64 `json:"voltage"`
	Current         float64 `json:"current"`
	PowerUsage      float64 `json:"power_usage"`
	Timestamp       string  `json:"timestamp"`
}

// ChaosCommand represents the instruction sent to a sensor to simulate a failure.
type ChaosCommand struct {
	Command   string `json:"command"`
	Duration  string `json:"duration"`
	Intensity string `json:"intensity"`
}

// ChaosController handles the periodic injection of chaos into the sensor fleet.
type ChaosController struct {
	MqttBroker string
	Namespace  string

	randMu     sync.Mutex
	randSource *rand.Rand
}

// Run starts the chaos injection loop.
func (c *ChaosController) Run(ctx context.Context) error {
	// 1. Initialize Kubernetes Client
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to get in-cluster config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// 2. Initialize MQTT Client
	opts := mqtt.NewClientOptions().AddBroker(c.MqttBroker)
	opts.SetClientID("chaos-controller")
	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT: %w", token.Error())
	}
	defer client.Disconnect(250)

	log.Printf("Chaos Controller started. Targeting namespace: %s", c.Namespace)

	for {
		// Randomize interval between 15s and 45s (Average 30s)
		interval := time.Duration(15+c.randIntn(31)) * time.Second
		timer := time.NewTimer(interval)

		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
			c.injectChaos(ctx, clientset, client)
		}
	}
}

func (c *ChaosController) injectChaos(ctx context.Context, k8s kubernetes.Interface, mqttClient mqtt.Client) {
	pods, err := k8s.CoreV1().Pods(c.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=sensor-fleet",
	})
	if err != nil {
		log.Printf("Error listing pods: %v", err)
		return
	}

	if len(pods.Items) == 0 {
		log.Println("No sensor pods found to target.")
		return
	}

	// Target a random pod
	targetPod := pods.Items[c.randIntn(len(pods.Items))]

	// Randomize Spike Parameters
	durationSec := 10 + c.randIntn(21) // 10s to 30s
	intensity := []string{"low", "medium", "high"}[c.randIntn(3)]

	log.Printf("Injecting Chaos into %s: Intensity=%s, Duration=%ds", targetPod.Name, intensity, durationSec)

	topic := fmt.Sprintf("sensors/%s/chaos", targetPod.Name)
	payload := fmt.Sprintf(`{"command": "spike", "duration": "%ds", "intensity": "%s"}`, durationSec, intensity)

	token := mqttClient.Publish(topic, 1, false, payload)
	token.Wait()
	if token.Error() != nil {
		log.Printf("Error publishing chaos command to %s: %v", targetPod.Name, token.Error())
	}
}

// Sensor represents a synthetic hardware sensor.
type Sensor struct {
	ID              string
	DeviceID        string
	FirmwareVersion string
	MqttBroker      string
	TelemetryTopic  string

	mu             sync.Mutex
	isSpiking      bool
	spikeIntensity string
	randMu         sync.Mutex
	randSource     *rand.Rand
}

// Run starts the sensor data generation and chaos subscription loop.
func (s *Sensor) Run(ctx context.Context) error {
	opts := mqtt.NewClientOptions().AddBroker(s.MqttBroker)
	opts.SetClientID(s.ID)
	opts.SetCleanSession(true)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(5 * time.Second)
	opts.SetAutoReconnect(true)

	// --- Chaos Subscription Logic ---
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Printf("Connected to MQTT broker at %s", s.MqttBroker)
		topic := fmt.Sprintf("sensors/%s/chaos", s.ID)
		if token := c.Subscribe(topic, 1, s.handleChaos); token.Wait() && token.Error() != nil {
			log.Printf("Error subscribing to chaos topic: %v", token.Error())
		} else {
			log.Printf("Subscribed to chaos topic: %s", topic)
		}
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return fmt.Errorf("failed to connect to MQTT: %w", token.Error())
	}
	defer client.Disconnect(250)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	telemetryTopic := s.telemetryTopic()
	log.Printf("Sensor %s started publishing to %s...", s.ID, telemetryTopic)

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			data := s.generateData()
			payload, err := json.Marshal(data)
			if err != nil {
				log.Printf("Error marshaling data: %v", err)
				continue
			}

			token := client.Publish(telemetryTopic, 1, false, payload)
			token.Wait()
			if token.Error() != nil {
				log.Printf("Error publishing to MQTT: %v", token.Error())
			}
		}
	}
}

func (s *Sensor) handleChaos(client mqtt.Client, msg mqtt.Message) {
	var cmd ChaosCommand
	if err := json.Unmarshal(msg.Payload(), &cmd); err != nil {
		log.Printf("Error unmarshaling chaos command: %v", err)
		return
	}

	if cmd.Command == "spike" {
		duration, _ := time.ParseDuration(cmd.Duration)
		if duration == 0 {
			duration = 10 * time.Second
		}

		log.Printf("!!! Chaos Spike Started: %s duration (Intensity: %s) !!!", duration, cmd.Intensity)
		s.mu.Lock()
		s.isSpiking = true
		s.spikeIntensity = cmd.Intensity
		s.mu.Unlock()

		time.AfterFunc(duration, func() {
			s.mu.Lock()
			s.isSpiking = false
			s.spikeIntensity = ""
			s.mu.Unlock()
			log.Println("Chaos Spike Ended.")
		})
	}
}

func (s *Sensor) generateData() SensorData {
	s.mu.Lock()
	spiking := s.isSpiking
	intensity := s.spikeIntensity
	s.mu.Unlock()

	// Base Simulation (Healthy state)
	temp := 35.0 + s.randFloat64()*5.0
	voltage := 5.0 - s.randFloat64()*0.1
	current := 0.4 + s.randFloat64()*0.4

	// Apply Dynamic Spike Logic
	if spiking {
		var tempAdd, currentAdd, voltageSag float64
		switch intensity {
		case "low":
			tempAdd = 5.0 + s.randFloat64()*5.0
			currentAdd = 0.6 + s.randFloat64()*0.6
			voltageSag = 0.2 + s.randFloat64()*0.2
		case "medium":
			tempAdd = 15.0 + s.randFloat64()*10.0
			currentAdd = 3.0 + s.randFloat64()*2.0
			voltageSag = 0.6 + s.randFloat64()*0.4
		case "high":
			tempAdd = 30.0 + s.randFloat64()*20.0
			currentAdd = 8.0 + s.randFloat64()*4.0
			voltageSag = 1.1 + s.randFloat64()*0.7
		default:
			tempAdd = 10.0
			currentAdd = 1.0
			voltageSag = 0.4
		}
		temp += tempAdd
		current += currentAdd
		voltage -= voltageSag
	}

	// Try to read actual temperature if available (HostPath mount)
	if b, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp"); err == nil {
		var t int
		if _, err := fmt.Sscanf(string(b), "%d", &t); err == nil {
			temp = float64(t) / 1000.0
			if spiking {
				// Relative spike for physical data
				temp += 10.0
			}
		}
	}

	return SensorData{
		SensorID:        s.ID,
		DeviceID:        s.deviceID(),
		FirmwareVersion: s.firmwareVersion(),
		TelemetryTopic:  s.telemetryTopic(),
		Temperature:     temp,
		Voltage:         voltage,
		Current:         current,
		PowerUsage:      voltage * current,
		Timestamp:       time.Now().Format(time.RFC3339),
	}
}

func (s *Sensor) deviceID() string {
	if s.DeviceID != "" {
		return s.DeviceID
	}
	return s.ID
}

func (s *Sensor) firmwareVersion() string {
	if s.FirmwareVersion != "" {
		return s.FirmwareVersion
	}
	return DefaultFirmwareVersion
}

func (s *Sensor) telemetryTopic() string {
	if s.TelemetryTopic != "" {
		return s.TelemetryTopic
	}
	return DefaultThermalTelemetryTopic
}

func (c *ChaosController) randIntn(n int) int {
	c.randMu.Lock()
	defer c.randMu.Unlock()
	if c.randSource != nil {
		return c.randSource.Intn(n)
	}
	return rand.Intn(n)
}

func (s *Sensor) randFloat64() float64 {
	s.randMu.Lock()
	defer s.randMu.Unlock()
	if s.randSource != nil {
		return s.randSource.Float64()
	}
	return rand.Float64()
}
