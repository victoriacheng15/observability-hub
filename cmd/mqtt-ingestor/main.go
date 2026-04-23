package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"observability-hub/internal/env"
	"observability-hub/internal/mqttingestor"
	"observability-hub/internal/telemetry"
)

const serviceName = "mqtt-ingestor"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	env.Load()

	shutdown, err := telemetry.Init(ctx, serviceName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize telemetry: %v\n", err)
		os.Exit(1)
	}
	defer shutdown()

	staleAfter, err := staleAfterFromEnv()
	if err != nil {
		telemetry.Error("mqtt_ingestor_invalid_stale_after", "error", err)
		os.Exit(1)
	}

	runtime, err := mqttingestor.NewRuntime(mqttingestor.RuntimeConfig{
		BrokerURL:             os.Getenv("MQTT_BROKER"),
		ClientID:              os.Getenv("MQTT_CLIENT_ID"),
		Environment:           os.Getenv("SIMULATION_ENV"),
		ExpectedSchemaVersion: os.Getenv("MQTT_SCHEMA_VERSION"),
		StaleAfter:            staleAfter,
	})
	if err != nil {
		telemetry.Error("mqtt_ingestor_init_failed", "error", err)
		os.Exit(1)
	}

	if err := runtime.Run(ctx); err != nil {
		telemetry.Error("mqtt_ingestor_run_failed", "error", err)
		os.Exit(1)
	}
}

func staleAfterFromEnv() (time.Duration, error) {
	raw := os.Getenv("MQTT_STALE_AFTER")
	if raw == "" {
		return 0, nil
	}

	duration, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("parse MQTT_STALE_AFTER: %w", err)
	}

	return duration, nil
}
