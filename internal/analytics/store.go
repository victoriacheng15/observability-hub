package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"observability-hub/internal/db/postgres"
)

const (
	tableSystemMetrics    = "system_metrics"
	tableAnalyticsMetrics = "analytics_metrics"
)

// MetricKind represents the type of analytics metric.
type MetricKind string

const (
	KindEnergy     MetricKind = "energy"
	KindCost       MetricKind = "cost"
	KindCarbon     MetricKind = "carbon"
	KindValueUnit  MetricKind = "value_unit"
	KindEfficiency MetricKind = "efficiency"
)

// MetricsStore handles persistence for consolidated host metrics in PostgreSQL.
type MetricsStore struct {
	Wrapper *postgres.PostgresWrapper
}

// NewMetricsStore creates a new MetricsStore.
func NewMetricsStore(w *postgres.PostgresWrapper) *MetricsStore {
	return &MetricsStore{Wrapper: w}
}

// EnsureSchema initializes the metrics tables and TimescaleDB hypertables if available.
func (s *MetricsStore) EnsureSchema(ctx context.Context) error {
	// 1. Legacy System Metrics
	querySystem := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			time TIMESTAMPTZ(0) NOT NULL,
			host TEXT NOT NULL,
			os TEXT NOT NULL,
			metric_type TEXT NOT NULL,
			payload JSONB NOT NULL
		);
	`, tableSystemMetrics)

	if _, err := s.Wrapper.Exec(ctx, "db.postgres.ensure_system_metrics", querySystem); err != nil {
		return fmt.Errorf("system_schema_init_failed: %w", err)
	}

	// 2. New Resource-to-Value Analytics Metrics
	queryEnum := `
		DO $$ BEGIN
			CREATE TYPE metric_kind AS ENUM ('energy', 'cost', 'carbon', 'value_unit', 'efficiency');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;
	`
	if _, err := s.Wrapper.Exec(ctx, "db.postgres.ensure_metric_kind_enum", queryEnum); err != nil {
		return fmt.Errorf("enum_init_failed: %w", err)
	}

	queryAnalytics := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			time        TIMESTAMPTZ NOT NULL,
			feature_id  TEXT NOT NULL,
			kind        metric_kind NOT NULL,
			value       DOUBLE PRECISION NOT NULL,
			unit        TEXT NOT NULL,
			metadata    JSONB DEFAULT '{}'
		);
	`, tableAnalyticsMetrics)

	if _, err := s.Wrapper.Exec(ctx, "db.postgres.ensure_analytics_metrics", queryAnalytics); err != nil {
		return fmt.Errorf("analytics_schema_init_failed: %w", err)
	}

	// 3. Enable hypertables if TimescaleDB is available
	queryHyperSystem := fmt.Sprintf("SELECT create_hypertable('%s', 'time', if_not_exists => true);", tableSystemMetrics)
	_, _ = s.Wrapper.Exec(ctx, "db.postgres.create_hypertable_system", queryHyperSystem)

	queryHyperAnalytics := fmt.Sprintf("SELECT create_hypertable('%s', 'time', if_not_exists => true);", tableAnalyticsMetrics)
	_, _ = s.Wrapper.Exec(ctx, "db.postgres.create_hypertable_analytics", queryHyperAnalytics)

	return nil
}

// RecordMetric inserts a single legacy system metric into the database.
func (s *MetricsStore) RecordMetric(ctx context.Context, t time.Time, hostName, osName, metricType string, payload interface{}) error {
	if payload == nil {
		return nil
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	query := fmt.Sprintf("INSERT INTO %s (time, host, os, metric_type, payload) VALUES ($1, $2, $3, $4, $5)", tableSystemMetrics)
	_, err = s.Wrapper.Exec(ctx, "db.postgres.record_legacy_metric", query, t, hostName, osName, metricType, payloadJSON)
	return err
}

// RecordAnalyticsMetric inserts a single resource-to-value metric into the database.
func (s *MetricsStore) RecordAnalyticsMetric(ctx context.Context, t time.Time, featureID string, kind MetricKind, value float64, unit string, metadata map[string]interface{}) error {
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	query := fmt.Sprintf("INSERT INTO %s (time, feature_id, kind, value, unit, metadata) VALUES ($1, $2, $3, $4, $5, $6)", tableAnalyticsMetrics)
	_, err = s.Wrapper.Exec(ctx, "db.postgres.record_analytics_metric", query, t, featureID, string(kind), value, unit, metaJSON)
	return err
}
