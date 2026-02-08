package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// MetricsStore handles persistence for system metrics in PostgreSQL.
type MetricsStore struct {
	DB *sql.DB
}

// NewMetricsStore creates a new MetricsStore.
func NewMetricsStore(db *sql.DB) *MetricsStore {
	return &MetricsStore{DB: db}
}

// EnsureSchema initializes the system_metrics table and TimescaleDB hypertable if available.
func (s *MetricsStore) EnsureSchema(ctx context.Context) error {
	_, err := s.DB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS system_metrics (
			time TIMESTAMPTZ(0) NOT NULL,
			host TEXT NOT NULL,
			os TEXT NOT NULL,
			metric_type TEXT NOT NULL,
			payload JSONB NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("schema_init_failed: %w", err)
	}

	// Enable hypertable if TimescaleDB is available
	_, err = s.DB.ExecContext(ctx, "SELECT create_hypertable('system_metrics', 'time', if_not_exists => true);")
	if err != nil {
		slog.Info("hypertable_check", "status", "skipped_or_failed", "detail", err)
	}
	return nil
}

// RecordMetric inserts a single metric into the database.
func (s *MetricsStore) RecordMetric(ctx context.Context, t time.Time, hostName, osName, metricType string, payload interface{}) error {
	if payload == nil {
		return nil
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	_, err = s.DB.ExecContext(ctx,
		"INSERT INTO system_metrics (time, host, os, metric_type, payload) VALUES ($1, $2, $3, $4, $5)",
		t, hostName, osName, metricType, payloadJSON,
	)
	return err
}
