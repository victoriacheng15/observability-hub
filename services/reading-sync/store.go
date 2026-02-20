package main

import (
	"context"
	"fmt"
	"time"

	"db/mongodb"
	"db/postgres"
)

const (
	// Postgres Table Names
	tableReadingAnalytics = "reading_analytics"
	tableSyncHistory      = "reading_sync_history"

	// MongoDB Names
	mongoDatabase   = "reading-analytics"
	mongoCollection = "articles"
)

// ReadingStore handles persistence for reading analytics and sync history in PostgreSQL.
type ReadingStore struct {
	Wrapper *postgres.PostgresWrapper
}

// NewReadingStore creates a new ReadingStore.
func NewReadingStore(w *postgres.PostgresWrapper) *ReadingStore {
	return &ReadingStore{Wrapper: w}
}

// ReadingDocument represents a normalized reading event for external use.
type ReadingDocument struct {
	ID        string                 `json:"id" bson:"_id"`
	Source    string                 `json:"source" bson:"source"`
	Type      string                 `json:"event_type" bson:"event_type"`
	Timestamp interface{}            `json:"timestamp" bson:"timestamp"`
	Payload   map[string]interface{} `json:"payload" bson:"payload"`
	Meta      map[string]interface{} `json:"meta" bson:"meta"`
}

// EnsureSchema initializes the necessary tables for reading analytics.
func (s *ReadingStore) EnsureSchema(ctx context.Context) error {
	// Ensure reading_analytics table
	queryAnalytics := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id SERIAL PRIMARY KEY,
		mongo_id TEXT UNIQUE NOT NULL,
		event_timestamp TIMESTAMPTZ,
		source TEXT,
		event_type TEXT,
		payload JSONB,
		meta JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`, tableReadingAnalytics)

	_, err := s.Wrapper.Exec(ctx, "db.postgres.ensure_reading_analytics", queryAnalytics)
	if err != nil {
		return fmt.Errorf("failed to ensure %s table: %w", tableReadingAnalytics, err)
	}

	// Ensure reading_sync_history table
	queryHistory := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id SERIAL PRIMARY KEY,
		start_time TIMESTAMPTZ NOT NULL,
		end_time TIMESTAMPTZ NOT NULL,
		status TEXT NOT NULL,
		processed_count INTEGER NOT NULL,
		error_message TEXT,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`, tableSyncHistory)

	_, err = s.Wrapper.Exec(ctx, "db.postgres.ensure_reading_sync_history", queryHistory)
	if err != nil {
		return fmt.Errorf("failed to ensure %s table: %w", tableSyncHistory, err)
	}

	return nil
}

// RecordSyncHistory logs the outcome of a synchronization run.
func (s *ReadingStore) RecordSyncHistory(ctx context.Context, startTime, endTime time.Time, status string, processedCount int, errorMessage string) error {
	query := fmt.Sprintf(`INSERT INTO %s (start_time, end_time, status, processed_count, error_message)
		VALUES ($1, $2, $3, $4, $5)`, tableSyncHistory)

	_, err := s.Wrapper.Exec(ctx, "db.postgres.record_sync_history", query, startTime, endTime, status, processedCount, errorMessage)
	return err
}

// InsertReadingAnalytics inserts a single processed reading event into Postgres.
func (s *ReadingStore) InsertReadingAnalytics(ctx context.Context, mongoID string, timestamp interface{}, source, eventType string, payloadJSON, metaJSON []byte) error {
	query := fmt.Sprintf(`INSERT INTO %s (mongo_id, event_timestamp, source, event_type, payload, meta, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())
		 ON CONFLICT (mongo_id) DO NOTHING`, tableReadingAnalytics)

	_, err := s.Wrapper.Exec(ctx, "db.postgres.insert_reading_analytics", query, mongoID, timestamp, source, eventType, payloadJSON, metaJSON)
	return err
}

// --- MongoDB Logic ---

type MongoStoreWrapper struct {
	Wrapper *mongodb.MongoStore
}

func (m *MongoStoreWrapper) FetchIngestedArticles(ctx context.Context, limit int64) ([]ReadingDocument, error) {
	var docs []ReadingDocument
	filter := map[string]any{"status": "ingested"}
	err := m.Wrapper.Find(ctx, "db.mongodb.fetch_ingested_articles", mongoDatabase, mongoCollection, filter, &docs, limit)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

func (m *MongoStoreWrapper) MarkArticleAsProcessed(ctx context.Context, id string) error {
	update := map[string]any{"$set": map[string]any{"status": "processed"}}
	return m.Wrapper.UpdateByID(ctx, "db.mongodb.mark_article_processed", mongoDatabase, mongoCollection, id, update)
}

func (m *MongoStoreWrapper) Close(ctx context.Context) error {
	return m.Wrapper.Close(ctx)
}
