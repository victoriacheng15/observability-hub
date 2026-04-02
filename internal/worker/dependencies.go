package worker

import (
	"context"
	"fmt"
	"os"

	"observability-hub/internal/db/postgres"
	"observability-hub/internal/secrets"
	"observability-hub/internal/telemetry"
	"observability-hub/internal/worker/store"
)

// Dependencies encapsulates all shared resources required by the worker tasks.
type Dependencies struct {
	SecretStore secrets.SecretStore
	Store       *store.Store
}

// InitDependencies initializes the shared resource pool (OpenBao and Postgres).
func InitDependencies(ctx context.Context) (*Dependencies, error) {
	// 1. Initialize OpenBao (Secret Store)
	secretStore, err := secrets.NewBaoProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize secret store: %w", err)
	}

	// 2. Initialize Postgres and Wrap in Unified Store
	pgWrapper, err := postgres.ConnectPostgres("postgres", secretStore)
	if err != nil {
		telemetry.Error("postgres_initialization_failed", "error", err)
		return nil, fmt.Errorf("postgres_init_failed: %w", err)
	}

	workerStore := store.NewStore(pgWrapper)

	// 3. Ensure Schema is ready
	if err := workerStore.EnsureSchema(ctx); err != nil {
		telemetry.Error("schema_initialization_failed", "error", err)
		return nil, fmt.Errorf("schema_init_failed: %w", err)
	}

	return &Dependencies{
		SecretStore: secretStore,
		Store:       workerStore,
	}, nil
}

// Close ensures all underlying resources are gracefully released.
func (d *Dependencies) Close() {
	if d.Store != nil && d.Store.Wrapper != nil && d.Store.Wrapper.DB != nil {
		d.Store.Wrapper.DB.Close()
	}
	if d.SecretStore != nil {
		d.SecretStore.Close()
	}
}

// GetThanosURL returns the configured Thanos URL from environment.
func (d *Dependencies) GetThanosURL() string {
	return os.Getenv("THANOS_URL")
}
