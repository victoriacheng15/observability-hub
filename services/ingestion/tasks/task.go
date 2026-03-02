package tasks

import (
	"context"

	"db/postgres"
	"secrets"
)

// Task is the interface that all ingestion tasks must implement.
type Task interface {
	Name() string
	Run(ctx context.Context, db *postgres.PostgresWrapper, secrets secrets.SecretStore) error
}
