package main

import (
	"context"
	"os"

	"db/postgres"
	"env"
	"ingestion/tasks"
	"secrets"
	"telemetry"
)

// DummyTask is a placeholder task for testing the engine.
type DummyTask struct {
	TaskName string
}

// Name returns the name of the dummy task.
func (t *DummyTask) Name() string {
	return t.TaskName
}

// Run executes the dummy task.
func (t *DummyTask) Run(ctx context.Context, db *postgres.PostgresWrapper, secrets secrets.SecretStore) error {
	telemetry.Info("dummy_task_running", "task", t.Name())
	// In a real task, you would perform ingestion logic here.
	return nil
}

func main() {
	ctx := context.Background()
	env.Load()

	// 1. Initialize Telemetry
	shutdown, err := telemetry.Init(ctx, "ingestion.service")
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

	// 4. Define and Run Ingestion Tasks
	registeredTasks := []tasks.Task{
		&DummyTask{TaskName: "reading"},
		&DummyTask{TaskName: "brain"},
	}

	telemetry.Info("starting_ingestion_tasks", "task_count", len(registeredTasks))

	for _, task := range registeredTasks {
		if err := RunTask(ctx, task, pgWrapper, secretStore); err != nil {
			// The error is already logged within RunTask.
			// Depending on requirements, you might want to stop all tasks if one fails.
			// For now, we continue to the next task.
		}
	}

	telemetry.Info("all_ingestion_tasks_finished")
}
