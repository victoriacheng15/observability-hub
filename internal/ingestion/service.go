package ingestion

import (
	"context"
	"fmt"

	"observability-hub/internal/db/postgres"
	"observability-hub/internal/secrets"
	"observability-hub/internal/telemetry"
)

// App encapsulates the ingestion application logic.
type App struct {
	secretStore secrets.SecretStore
	pgWrapper   *postgres.PostgresWrapper
	tasks       []Task
}

// NewApp creates a new App instance.
func NewApp(secretStore secrets.SecretStore, pgWrapper *postgres.PostgresWrapper) *App {
	return &App{
		secretStore: secretStore,
		pgWrapper:   pgWrapper,
		tasks: []Task{
			&ReadingTask{},
			&BrainTask{},
		},
	}
}

// Run executes all registered ingestion tasks.
func (a *App) Run(ctx context.Context) error {
	telemetry.Info("starting_ingestion_tasks", "task_count", len(a.tasks))

	for _, task := range a.tasks {
		if err := RunTask(ctx, task, a.pgWrapper, a.secretStore); err != nil {
			telemetry.Error("task_failed", "task", fmt.Sprintf("%T", task), "error", err)
			// Depending on requirements, we might want to continue or stop.
			// Current behavior matches original: continue to next task.
		}
	}

	telemetry.Info("all_ingestion_tasks_finished")
	return nil
}
