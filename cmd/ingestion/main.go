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
	ctx := context.Background()
	env.Load()

	// 1. Initialize Telemetry
	shutdown, err := telemetry.Init(ctx, "ingestion")
	if err != nil {
		telemetry.Warn("otel_init_failed, continuing without full observability", "error", err)
	}

	// 2. Initialize Secret Store
	secretStore, err := secrets.NewBaoProvider()
	if err != nil {
		telemetry.Error("secret_provider_init_failed", "error", err)
		shutdown()
		os.Exit(1)
	}
	defer secretStore.Close()

	// 3. Connect to Postgres
	pgWrapper, err := postgres.ConnectPostgres("postgres", secretStore)
	if err != nil {
		telemetry.Error("postgres_connection_failed", "error", err)
		shutdown()
		os.Exit(1)
	}
	defer pgWrapper.DB.Close()

	// 4. Define and Run Ingestion Tasks
	registeredTasks := []ingestion.Task{
		&ingestion.ReadingTask{},
		&ingestion.BrainTask{},
	}

	telemetry.Info("starting_ingestion_tasks", "task_count", len(registeredTasks))

	for _, task := range registeredTasks {
		if err := ingestion.RunTask(ctx, task, pgWrapper, secretStore); err != nil {
			// The error is already logged within RunTask.
			// Depending on requirements, you might want to stop all tasks if one fails.
			// For now, we continue to the next task.
		}
	}

	telemetry.Info("all_ingestion_tasks_finished")
	shutdown()
}
