package ingestion

import (
	"context"
	"fmt"

	"observability-hub/internal/secrets"
	"observability-hub/internal/telemetry"
	"observability-hub/internal/worker/store"
)

// App encapsulates the ingestion application logic.
type App struct {
	secretStore secrets.SecretStore
	store       *store.Store
	tasks       []Task
}

// NewApp creates a new App instance.
func NewApp(secretStore secrets.SecretStore, s *store.Store) *App {
	return &App{
		secretStore: secretStore,
		store:       s,
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
		if err := RunTask(ctx, task, a.store, a.secretStore); err != nil {
			telemetry.Error("task_failed", "task", fmt.Sprintf("%T", task), "error", err)
		}
	}

	telemetry.Info("all_ingestion_tasks_finished")
	return nil
}
