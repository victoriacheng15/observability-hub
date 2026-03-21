package main

import (
	"context"
	"log"
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

	namespace := os.Getenv("TARGET_NAMESPACE")
	if namespace == "" {
		namespace = "hardware-sim"
	}

	controller := &hardwaresim.ChaosController{
		MqttBroker: mqttBroker,
		Namespace:  namespace,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		cancel()
	}()

	if err := controller.Run(ctx); err != nil {
		log.Fatalf("Chaos Controller failed: %v", err)
	}
}
