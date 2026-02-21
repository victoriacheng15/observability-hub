package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"db/postgres"
)

const (
	tableSystemMetrics = "system_metrics"
)

// MetricsStore handles persistence for system metrics in PostgreSQL.
type MetricsStore struct {
	Wrapper *postgres.PostgresWrapper
}

// NewMetricsStore creates a new MetricsStore.
func NewMetricsStore(w *postgres.PostgresWrapper) *MetricsStore {
	return &MetricsStore{Wrapper: w}
}

// EnsureSchema initializes the system_metrics table and TimescaleDB hypertable if available.
func (s *MetricsStore) EnsureSchema(ctx context.Context) error {
	queryTable := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			time TIMESTAMPTZ(0) NOT NULL,
			host TEXT NOT NULL,
			os TEXT NOT NULL,
			metric_type TEXT NOT NULL,
			payload JSONB NOT NULL
		);
	`, tableSystemMetrics)

	_, err := s.Wrapper.Exec(ctx, "db.postgres.ensure_system_metrics", queryTable)
	if err != nil {
		return fmt.Errorf("schema_init_failed: %w", err)
	}

	// Enable hypertable if TimescaleDB is available
	queryHyper := fmt.Sprintf("SELECT create_hypertable('%s', 'time', if_not_exists => true);", tableSystemMetrics)
	_, _ = s.Wrapper.Exec(ctx, "db.postgres.create_hypertable", queryHyper)

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

	query := fmt.Sprintf("INSERT INTO %s (time, host, os, metric_type, payload) VALUES ($1, $2, $3, $4, $5)", tableSystemMetrics)
	_, err = s.Wrapper.Exec(ctx, "db.postgres.record_metric", query, t, hostName, osName, metricType, payloadJSON)
	return err
}
