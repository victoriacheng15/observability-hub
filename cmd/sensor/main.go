package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"

	"observability-hub/internal/hardware-sim"
)

func main() {
	mqttBroker := os.Getenv("MQTT_BROKER")
	if mqttBroker == "" {
		mqttBroker = "tcp://emqx.observability:1883"
	}

	sensorID := os.Getenv("HOSTNAME")
	if sensorID == "" {
		sensorID = fmt.Sprintf("sensor-%d", rand.Intn(1000))
	}

	deviceID := os.Getenv("DEVICE_ID")
	if deviceID == "" {
		deviceID = sensorID
	}

	firmwareVersion := os.Getenv("FIRMWARE_VERSION")
	telemetryTopic := os.Getenv("TELEMETRY_TOPIC")

	s := &hardwaresim.Sensor{
		ID:              sensorID,
		DeviceID:        deviceID,
		FirmwareVersion: firmwareVersion,
		MqttBroker:      mqttBroker,
		TelemetryTopic:  telemetryTopic,
	}

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

	if err := s.Run(ctx); err != nil {
		log.Fatalf("Sensor failed: %v", err)
	}
}
