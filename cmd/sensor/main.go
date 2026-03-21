package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// SensorData represents the synthetic sensor payload.
type SensorData struct {
	SensorID    string  `json:"sensor_id"`
	Temperature float64 `json:"temperature"`
	PowerUsage  float64 `json:"power_usage"`
	Timestamp   string  `json:"timestamp"`
}

type ChaosCommand struct {
	Command   string `json:"command"`
	Duration  string `json:"duration"`
	Intensity string `json:"intensity"`
}

var (
	isSpiking      bool
	spikeIntensity string
	spikeMu        sync.Mutex
)

func main() {
	// ... rest of main ...
	mqttBroker := os.Getenv("MQTT_BROKER")
	if mqttBroker == "" {
		mqttBroker = "tcp://emqx.observability:1883"
	}

	sensorID := os.Getenv("HOSTNAME")
	if sensorID == "" {
		sensorID = fmt.Sprintf("sensor-%d", rand.Intn(1000))
	}

	opts := mqtt.NewClientOptions().AddBroker(mqttBroker)
	opts.SetClientID(sensorID)
	opts.SetCleanSession(true)
	opts.SetConnectTimeout(10 * time.Second)
	opts.SetKeepAlive(30 * time.Second)
	opts.SetPingTimeout(5 * time.Second)
	opts.SetAutoReconnect(true)
	// --- Chaos Subscription Logic ---
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		log.Printf("Connected to MQTT broker at %s", mqttBroker)
		// Use the base HOSTNAME for the chaos topic to allow consistent targeting
		baseID := os.Getenv("HOSTNAME")
		if baseID == "" {
			baseID = sensorID
		}
		topic := fmt.Sprintf("sensors/%s/chaos", baseID)
		if token := c.Subscribe(topic, 1, handleChaos); token.Wait() && token.Error() != nil {
			log.Printf("Error subscribing to chaos topic: %v", token.Error())
		} else {
			log.Printf("Subscribed to chaos topic: %s", topic)
		}
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT: %v", token.Error())
	}
	defer client.Disconnect(250)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Signal handling for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("Shutting down sensor...")
		cancel()
	}()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Printf("Sensor %s started publishing to sensors/thermal...", sensorID)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data := generateData(sensorID)
			payload, err := json.Marshal(data)
			if err != nil {
				log.Printf("Error marshaling data: %v", err)
				continue
			}

			token := client.Publish("sensors/thermal", 1, false, payload)
			token.Wait()
			if token.Error() != nil {
				log.Printf("Error publishing to MQTT: %v", token.Error())
			}
		}
	}
}

func handleChaos(client mqtt.Client, msg mqtt.Message) {
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
		spikeMu.Lock()
		isSpiking = true
		spikeIntensity = cmd.Intensity
		spikeMu.Unlock()

		time.AfterFunc(duration, func() {
			spikeMu.Lock()
			isSpiking = false
			spikeIntensity = ""
			spikeMu.Unlock()
			log.Println("Chaos Spike Ended.")
		})
	}
}

func generateData(id string) SensorData {
	spikeMu.Lock()
	spiking := isSpiking
	intensity := spikeIntensity
	spikeMu.Unlock()

	// Base Simulation (Healthy state)
	temp := 35.0 + rand.Float64()*5.0
	power := 2.0 + rand.Float64()*2.0

	// Apply Dynamic Spike Logic
	if spiking {
		var tempAdd, powerAdd float64
		switch intensity {
		case "low":
			tempAdd = 5.0 + rand.Float64()*5.0
			powerAdd = 2.0 + rand.Float64()*3.0
		case "medium":
			tempAdd = 15.0 + rand.Float64()*10.0
			powerAdd = 10.0 + rand.Float64()*10.0
		case "high":
			tempAdd = 30.0 + rand.Float64()*20.0
			powerAdd = 25.0 + rand.Float64()*20.0
		default:
			tempAdd = 10.0
			powerAdd = 5.0
		}
		temp += tempAdd
		power += powerAdd
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
		SensorID:    id,
		Temperature: temp,
		PowerUsage:  power,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}
