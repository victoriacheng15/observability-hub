package ingestion

import (
	"context"
	"fmt"
	"time"

	"observability-hub/internal/db/postgres"
)

// --- Generic Sync History Helpers ---

// RecordSyncHistory is a generic helper to log the result of any ingestion task.
func RecordSyncHistory(ctx context.Context, wrapper *postgres.PostgresWrapper, tableName string, startTime, endTime time.Time, status string, processedCount int, errorMessage string) error {
	query := fmt.Sprintf(`INSERT INTO %s (start_time, end_time, status, processed_count, error_message)
		VALUES ($1, $2, $3, $4, $5)`, tableName)

	_, err := wrapper.Exec(ctx, "db.postgres.record_sync_history", query, startTime, endTime, status, processedCount, errorMessage)
	return err
}

// EnsureHistorySchema creates the standard sync history table for a given task.
func EnsureHistorySchema(ctx context.Context, wrapper *postgres.PostgresWrapper, tableName string) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY,
			start_time TIMESTAMPTZ NOT NULL,
			end_time TIMESTAMPTZ NOT NULL,
			status TEXT NOT NULL,
			processed_count INTEGER NOT NULL,
			error_message TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`, tableName)

	_, err := wrapper.Exec(ctx, "db.postgres.ensure_sync_history", query)
	return err
}

// --- Reading Task Store ---

const (
	TableReadingAnalytics = "reading_analytics"
	TableReadingSync      = "reading_sync_history"
	MongoDatabase         = "reading-analytics"
	MongoCollection       = "articles"
)

type ReadingStore struct {
	Wrapper *postgres.PostgresWrapper
}

func NewReadingStore(w *postgres.PostgresWrapper) *ReadingStore {
	return &ReadingStore{Wrapper: w}
}

type ReadingDocument struct {
	ID        string                 `json:"id" bson:"_id"`
	Source    string                 `json:"source" bson:"source"`
	Type      string                 `json:"event_type" bson:"event_type"`
	Timestamp interface{}            `json:"timestamp" bson:"timestamp"`
	Payload   map[string]interface{} `json:"payload" bson:"payload"`
	Meta      map[string]interface{} `json:"meta" bson:"meta"`
}

func (s *ReadingStore) EnsureSchema(ctx context.Context) error {
	queryAnalytics := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id SERIAL PRIMARY KEY,
		mongo_id TEXT UNIQUE NOT NULL,
		event_timestamp TIMESTAMPTZ,
		source TEXT,
		event_type TEXT,
		payload JSONB,
		meta JSONB,
		created_at TIMESTAMPTZ DEFAULT NOW()
	)`, TableReadingAnalytics)

	_, err := s.Wrapper.Exec(ctx, "db.postgres.ensure_reading_analytics", queryAnalytics)
	if err != nil {
		return fmt.Errorf("failed to ensure %s table: %w", TableReadingAnalytics, err)
	}

	return EnsureHistorySchema(ctx, s.Wrapper, TableReadingSync)
}

func (s *ReadingStore) RecordSyncHistory(ctx context.Context, startTime, endTime time.Time, status string, processedCount int, errorMessage string) error {
	return RecordSyncHistory(ctx, s.Wrapper, TableReadingSync, startTime, endTime, status, processedCount, errorMessage)
}

func (s *ReadingStore) InsertReadingAnalytics(ctx context.Context, mongoID string, timestamp interface{}, source, eventType string, payloadJSON, metaJSON []byte) error {
	query := fmt.Sprintf(`INSERT INTO %s (mongo_id, event_timestamp, source, event_type, payload, meta, created_at) 
		 VALUES ($1, $2, $3, $4, $5, $6, NOW())
		 ON CONFLICT (mongo_id) DO NOTHING`, TableReadingAnalytics)

	_, err := s.Wrapper.Exec(ctx, "db.postgres.insert_reading_analytics", query, mongoID, timestamp, source, eventType, payloadJSON, metaJSON)
	return err
}

// --- Brain Task Store ---

const (
	TableSecondBrain = "second_brain"
	TableBrainSync   = "second_brain_sync_history"
)

type BrainStore struct {
	Wrapper *postgres.PostgresWrapper
}

func NewBrainStore(w *postgres.PostgresWrapper) *BrainStore {
	return &BrainStore{Wrapper: w}
}

func (s *BrainStore) EnsureSchema(ctx context.Context) error {
	queryBrain := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
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
		)`, TableSecondBrain)

	_, err := s.Wrapper.Exec(ctx, "db.postgres.ensure_second_brain", queryBrain)
	if err != nil {
		return fmt.Errorf("failed to ensure %s table: %w", TableSecondBrain, err)
	}

	return EnsureHistorySchema(ctx, s.Wrapper, TableBrainSync)
}

func (s *BrainStore) RecordSyncHistory(ctx context.Context, startTime, endTime time.Time, status string, processedCount int, errorMessage string) error {
	return RecordSyncHistory(ctx, s.Wrapper, TableBrainSync, startTime, endTime, status, processedCount, errorMessage)
}

func (s *BrainStore) GetLatestEntryDate(ctx context.Context) (string, error) {
	var latestDate string
	query := fmt.Sprintf("SELECT COALESCE(MAX(entry_date)::text, '1970-01-01') FROM %s", TableSecondBrain)
	err := s.Wrapper.QueryRow(ctx, "db.postgres.get_latest_entry_date", query).Scan(&latestDate)
	if err != nil {
		return "", err
	}
	return latestDate, nil
}

func (s *BrainStore) InsertThought(ctx context.Context, date, content, category string, tags []string, contextString, checksum string, tokenCount int) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (entry_date, content, category, origin_type, tags, context_string, checksum, token_count)
		VALUES ($1, $2, $3, 'journal', $4, $5, $6, $7)
		ON CONFLICT (checksum) DO NOTHING`, TableSecondBrain)

	_, err := s.Wrapper.Exec(ctx, "db.postgres.insert_thought", query,
		date, content, category, s.Wrapper.Array(tags), contextString, checksum, tokenCount)
	return err
}
