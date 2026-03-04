package main

import (
	"context"
	"fmt"

	"observability-hub/cmd/ingestion/tasks"
	"observability-hub/internal/db/postgres"
	"observability-hub/internal/secrets"
	"observability-hub/internal/telemetry"
)

// RunTask executes a single ingestion task, wrapping it with observability and error handling.
func RunTask(ctx context.Context, task tasks.Task, db *postgres.PostgresWrapper, secretStore secrets.SecretStore) error {
	tracer := telemetry.GetTracer("ingestion.engine")
	ctx, span := tracer.Start(ctx, fmt.Sprintf("task.%s", task.Name()))
	defer span.End()

	telemetry.Info("running_task", "task", task.Name())
	span.SetAttributes(telemetry.StringAttribute("task.name", task.Name()))

	err := task.Run(ctx, db, secretStore)
	if err != nil {
		telemetry.Error("task_failed", "task", task.Name(), "error", err)
		span.SetStatus(telemetry.CodeError, err.Error())
		return err
	}

	telemetry.Info("task_succeeded", "task", task.Name())
	span.SetStatus(telemetry.CodeOk, "success")
	return nil
}
