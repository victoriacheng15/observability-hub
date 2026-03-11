package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"observability-hub/internal/analytics"
	"observability-hub/internal/db/postgres"
	"observability-hub/internal/env"
	"observability-hub/internal/secrets"
	"observability-hub/internal/telemetry"
)

func main() {
	env.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	thanosURL := os.Getenv("THANOS_URL")
	if thanosURL == "" {
		telemetry.Error("missing_thanos_url")
		os.Exit(1)
	}

	// 1. Initialize Telemetry
	shutdown, err := telemetry.Init(ctx, analytics.ServiceName)
	if err != nil {
		fmt.Printf("Warning: OTel init failed: %v\n", err)
	}
	defer shutdown()

	// 2. Initialize Secrets & DB
	secretStore, err := secrets.NewBaoProvider()
	if err != nil {
		telemetry.Error("secret_provider_init_failed", "error", err)
		os.Exit(1)
	}

	wrapper, err := postgres.ConnectPostgres("postgres", secretStore)
	if err != nil {
		telemetry.Error("db_connection_failed", "error", err)
		os.Exit(1)
	}
	defer wrapper.DB.Close()

	store := analytics.NewMetricsStore(wrapper)
	if err := store.EnsureSchema(ctx); err != nil {
		telemetry.Error("schema_init_failed", "error", err)
		os.Exit(1)
	}

	thanosClient := analytics.NewThanosClient(thanosURL)
	svc := analytics.NewService(
		thanosClient,
		store,
		analytics.NewThanosResourceProvider(thanosClient),
	)

	if err := svc.Start(ctx); err != nil {
		telemetry.Error("service_failed", "error", err)
		os.Exit(1)
	}
}
