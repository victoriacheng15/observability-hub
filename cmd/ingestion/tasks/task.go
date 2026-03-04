package tasks

import (
	"context"

	"observability-hub/internal/db/postgres"
	"observability-hub/internal/secrets"
)

// Task is the interface that all ingestion tasks must implement.
type Task interface {
	Name() string
	Run(ctx context.Context, db *postgres.PostgresWrapper, secrets secrets.SecretStore) error
}
