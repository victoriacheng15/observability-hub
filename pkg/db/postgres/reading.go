package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ReadingStore handles persistence for reading analytics and sync history in PostgreSQL.
type ReadingStore struct {
	DB *sql.DB
}

// NewReadingStore creates a new ReadingStore.
func NewReadingStore(db *sql.DB) *ReadingStore {
	return &ReadingStore{DB: db}
}

// EnsureSchema initializes the necessary tables for reading analytics.
func (s *ReadingStore) EnsureSchema(ctx context.Context) error {
	// Ensure reading_analytics table
	_, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS reading_analytics (
		id SERIAL PRIMARY KEY,
		mongo_id TEXT UNIQUE NOT NULL,
		event_timestamp TIMESTAMPTZ,
		source TEXT,
		event_type TEXT,
		payload JSONB,
		meta JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`)
	if err != nil {
		return fmt.Errorf("failed to ensure reading_analytics table: %w", err)
	}

	// Ensure reading_sync_history table
	_, err = s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS reading_sync_history (
		id SERIAL PRIMARY KEY,
		start_time TIMESTAMPTZ NOT NULL,
		end_time TIMESTAMPTZ NOT NULL,
		status TEXT NOT NULL,
		processed_count INTEGER NOT NULL,
		error_message TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`)
	if err != nil {
		return fmt.Errorf("failed to ensure reading_sync_history table: %w", err)
	}

	return nil
}

// RecordSyncHistory logs the outcome of a synchronization run.
func (s *ReadingStore) RecordSyncHistory(ctx context.Context, startTime, endTime time.Time, status string, processedCount int, errorMessage string) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO reading_sync_history (start_time, end_time, status, processed_count, error_message)
		VALUES ($1, $2, $3, $4, $5)`,
		startTime, endTime, status, processedCount, errorMessage,
	)
	return err
}

// InsertReadingAnalytics inserts a single processed reading event into Postgres.
func (s *ReadingStore) InsertReadingAnalytics(ctx context.Context, mongoID string, timestamp interface{}, source, eventType string, payloadJSON, metaJSON []byte) error {
	_, err := s.DB.ExecContext(ctx,
		`INSERT INTO reading_analytics (mongo_id, event_timestamp, source, event_type, payload, meta, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())
		 ON CONFLICT (mongo_id) DO NOTHING`,
		mongoID, timestamp, source, eventType, payloadJSON, metaJSON,
	)
	return err
}
