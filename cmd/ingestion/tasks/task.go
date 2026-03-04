package tasks

import (
	"context"

	"observability-hub/pkg/db/postgres"
	"observability-hub/pkg/secrets"
)

// Task is the interface that all ingestion tasks must implement.
type Task interface {
	Name() string
	Run(ctx context.Context, db *postgres.PostgresWrapper, secrets secrets.SecretStore) error
}
