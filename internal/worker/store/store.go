package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"observability-hub/internal/db/postgres"
)

// --- Shared Types & Constants ---

type MetricKind string

const (
	KindEnergy     MetricKind = "energy"
	KindCost       MetricKind = "cost"
	KindCarbon     MetricKind = "carbon"
	KindValueUnit  MetricKind = "value_unit"
	KindEfficiency MetricKind = "efficiency"
)

const (
	TableAnalyticsMetrics = "analytics_metrics"
	TableReadingAnalytics = "reading_analytics"
	TableReadingSync      = "reading_sync_history"
	TableSecondBrain      = "second_brain"
	TableBrainSync        = "second_brain_sync_history"
)

type ReadingDocument struct {
	ID        string                 `json:"id" bson:"_id"`
	Source    string                 `json:"source" bson:"source"`
	Type      string                 `json:"event_type" bson:"event_type"`
	Timestamp interface{}            `json:"timestamp" bson:"timestamp"`
	Payload   map[string]interface{} `json:"payload" bson:"payload"`
	Meta      map[string]interface{} `json:"meta" bson:"meta"`
}

// Store handles all database persistence for the unified worker.
type Store struct {
	Wrapper *postgres.PostgresWrapper
}

func NewStore(w *postgres.PostgresWrapper) *Store {
	return &Store{Wrapper: w}
}

// --- Schema Management ---

func (s *Store) EnsureSchema(ctx context.Context) error {
	// 1. Analytics Metrics (Enum + Table)
	qEnum := `DO $$ BEGIN CREATE TYPE metric_kind AS ENUM ('energy', 'cost', 'carbon', 'value_unit', 'efficiency'); EXCEPTION WHEN duplicate_object THEN null; END $$;`
	if _, err := s.Wrapper.Exec(ctx, "db.ensure_metric_kind_enum", qEnum); err != nil {
		return err
	}

	q2 := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		time TIMESTAMPTZ NOT NULL,
		feature_id TEXT NOT NULL,
		kind metric_kind NOT NULL,
		value DOUBLE PRECISION NOT NULL,
		unit TEXT NOT NULL,
		metadata JSONB DEFAULT '{}'
	);`, TableAnalyticsMetrics)
	if _, err := s.Wrapper.Exec(ctx, "db.ensure_analytics_metrics", q2); err != nil {
		return err
	}

	// Add unique constraint for idempotency
	qIdx := fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS idx_analytics_metrics_idempotency ON %s (time, feature_id, kind);", TableAnalyticsMetrics)
	if _, err := s.Wrapper.Exec(ctx, "db.ensure_analytics_idempotency_idx", qIdx); err != nil {
		return err
	}

	// 2. Reading Analytics
	q3 := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id SERIAL PRIMARY KEY,
		mongo_id TEXT UNIQUE NOT NULL,
		event_timestamp TIMESTAMPTZ,
		source TEXT,
		event_type TEXT,
		payload JSONB,
		meta JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);`, TableReadingAnalytics)
	if _, err := s.Wrapper.Exec(ctx, "db.ensure_reading_analytics", q3); err != nil {
		return err
	}

	// 3. Second Brain
	q4 := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id SERIAL PRIMARY KEY,
		entry_date DATE NOT NULL,
		content TEXT NOT NULL,
		category TEXT,
		origin_type TEXT,
		tags TEXT[],
		context_string TEXT,
		checksum TEXT UNIQUE NOT NULL,
		token_count INTEGER,
		created_at TIMESTAMPTZ DEFAULT NOW()
	);`, TableSecondBrain)
	if _, err := s.Wrapper.Exec(ctx, "db.ensure_second_brain", q4); err != nil {
		return err
	}

	// 4. Sync History Tables
	tables := []string{TableReadingSync, TableBrainSync}
	for _, t := range tables {
		q := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY,
			start_time TIMESTAMPTZ NOT NULL,
			end_time TIMESTAMPTZ NOT NULL,
			status TEXT NOT NULL,
			processed_count INTEGER NOT NULL,
			error_message TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		);`, t)
		if _, err := s.Wrapper.Exec(ctx, "db.ensure_history_table", q); err != nil {
			return err
		}
	}

	// 5. Hypertables
	_, _ = s.Wrapper.Exec(ctx, "db.create_hypertable_analytics", fmt.Sprintf("SELECT create_hypertable('%s', 'time', if_not_exists => true);", TableAnalyticsMetrics))

	return nil
}

// --- Shared Recording Methods ---

func (s *Store) RecordSyncHistory(ctx context.Context, tableName string, startTime, endTime time.Time, status string, processedCount int, errorMessage string) error {
	query := fmt.Sprintf(`INSERT INTO %s (start_time, end_time, status, processed_count, error_message) VALUES ($1, $2, $3, $4, $5)`, tableName)
	_, err := s.Wrapper.Exec(ctx, "db.record_sync_history", query, startTime, endTime, status, processedCount, errorMessage)
	return err
}

// --- Analytics Methods ---

func (s *Store) RecordAnalyticsMetric(ctx context.Context, t time.Time, featureID string, kind MetricKind, value float64, unit string, metadata map[string]interface{}) error {
	mJSON, _ := json.Marshal(metadata)
	q := fmt.Sprintf(`INSERT INTO %s (time, feature_id, kind, value, unit, metadata) 
		 VALUES ($1, $2, $3, $4, $5, $6) 
		 ON CONFLICT (time, feature_id, kind) 
		 DO UPDATE SET value = EXCLUDED.value, metadata = EXCLUDED.metadata`, TableAnalyticsMetrics)
	_, err := s.Wrapper.Exec(ctx, "db.record_analytics_metric", q, t, featureID, string(kind), value, unit, mJSON)
	return err
}

// --- Ingestion Methods ---

func (s *Store) InsertReadingAnalytics(ctx context.Context, mongoID string, timestamp interface{}, source, eventType string, payloadJSON, metaJSON []byte) error {
	q := fmt.Sprintf(`INSERT INTO %s (mongo_id, event_timestamp, source, event_type, payload, meta, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, NOW()) ON CONFLICT (mongo_id) DO NOTHING`, TableReadingAnalytics)
	_, err := s.Wrapper.Exec(ctx, "db.insert_reading_analytics", q, mongoID, timestamp, source, eventType, payloadJSON, metaJSON)
	return err
}

func (s *Store) GetLatestEntryDate(ctx context.Context) (string, error) {
	var latestDate string
	q := fmt.Sprintf("SELECT COALESCE(MAX(entry_date)::text, '1970-01-01') FROM %s", TableSecondBrain)
	err := s.Wrapper.QueryRow(ctx, "db.get_latest_entry_date", q).Scan(&latestDate)
	return latestDate, err
}

func (s *Store) InsertThought(ctx context.Context, date, content, category string, tags []string, contextString, checksum string, tokenCount int) error {
	q := fmt.Sprintf(`INSERT INTO %s (entry_date, content, category, origin_type, tags, context_string, checksum, token_count)
		VALUES ($1, $2, $3, 'journal', $4, $5, $6, $7) ON CONFLICT (checksum) DO NOTHING`, TableSecondBrain)
	_, err := s.Wrapper.Exec(ctx, "db.insert_thought", q, date, content, category, s.Wrapper.Array(tags), contextString, checksum, tokenCount)
	return err
}
