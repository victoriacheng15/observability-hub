package main

import (
	"context"
	"os"

	"observability-hub/internal/db/postgres"
	"observability-hub/internal/env"
	"observability-hub/internal/ingestion"
	"observability-hub/internal/secrets"
	"observability-hub/internal/telemetry"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env.Load()

	// 1. Initialize Telemetry
	shutdown, err := telemetry.Init(ctx, "ingestion")
	if err != nil {
		telemetry.Warn("otel_init_failed, continuing without full observability", "error", err)
	}
	defer shutdown()

	// 2. Initialize Secret Store
	secretStore, err := secrets.NewBaoProvider()
	if err != nil {
		telemetry.Error("secret_provider_init_failed", "error", err)
		os.Exit(1)
	}
	defer secretStore.Close()

	// 3. Connect to Postgres
	pgWrapper, err := postgres.ConnectPostgres("postgres", secretStore)
	if err != nil {
		telemetry.Error("postgres_connection_failed", "error", err)
		os.Exit(1)
	}
	defer pgWrapper.DB.Close()

	// 4. Initialize and Run App
	app := ingestion.NewApp(secretStore, pgWrapper)
	if err := app.Run(ctx); err != nil {
		telemetry.Error("ingestion_failed", "error", err)
		os.Exit(1)
	}
}
