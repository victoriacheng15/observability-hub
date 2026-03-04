package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"observability-hub/internal/db/mongodb"
	"observability-hub/internal/db/postgres"
	"observability-hub/internal/secrets"
	"observability-hub/internal/telemetry"
)

// ReadingTask implements the Task interface for syncing reading analytics data.
type ReadingTask struct{}

var (
	readingOnce  sync.Once
	readingReady bool
	// Global metrics initialized once
	processedCounter telemetry.Int64Counter
)

func ensureReadingGlobalMetrics() {
	readingOnce.Do(func() {
		meterObj := telemetry.GetMeter("reading.sync")
		processedCounter, _ = telemetry.NewInt64Counter(meterObj, "reading.sync.processed.total", "Total documents processed")
		readingReady = true
	})
}

// Name returns the name of the task.
func (t *ReadingTask) Name() string {
	return "reading"
}

// Run executes the reading sync task.
func (t *ReadingTask) Run(ctx context.Context, db *postgres.PostgresWrapper, secretStore secrets.SecretStore) error {
	ensureReadingGlobalMetrics()
	readingStore := NewReadingStore(db)

	mongoStore, err := newMongoStore(secretStore)
	if err != nil {
		return fmt.Errorf("mongo_connection_failed: %w", err)
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := mongoStore.Close(closeCtx); err != nil {
			telemetry.Error("mongo_close_failed", "error", err)
		}
	}()

	return t.Sync(ctx, readingStore, mongoStore)
}

// Sync performs the data synchronization from MongoDB to PostgreSQL.
func (t *ReadingTask) Sync(ctx context.Context, pgStore *ReadingStore, mStore MongoStoreAPI) error {
	tracer := telemetry.GetTracer("reading.sync")
	meter := telemetry.GetMeter("reading.sync")

	// Task-specific metrics
	syncTotal, _ := telemetry.NewInt64Counter(meter, "reading.sync.total", "Total sync runs")
	errorsCounter, _ := telemetry.NewInt64Counter(meter, "reading.sync.errors.total", "Total sync errors")
	durationHist, _ := telemetry.NewInt64Histogram(meter, "reading.sync.duration.ms", "Sync duration in milliseconds", "ms")

	lastSuccessTime := time.Now()
	_, _ = telemetry.NewInt64ObservableGauge(meter, "reading.sync.lag.seconds", "Time since last successful sync", func(ctx context.Context, obs telemetry.Int64Observer) error {
		obs.Observe(int64(time.Since(lastSuccessTime).Seconds()))
		return nil
	})

	if syncTotal != nil {
		telemetry.AddInt64Counter(ctx, syncTotal, 1)
	}

	start := time.Now()
	ctx, span := tracer.Start(ctx, "job.reading_sync")
	defer span.End()

	startTime := time.Now().UTC()
	syncStatus := "success"
	var syncErrorMessage string
	processedCount := 0

	defer func() {
		if syncStatus == "success" {
			lastSuccessTime = time.Now()
		}
		if err := pgStore.RecordSyncHistory(ctx, startTime, time.Now().UTC(), syncStatus, processedCount, syncErrorMessage); err != nil {
			telemetry.Error("failed_to_record_sync_history", "error", err)
		}
	}()

	telemetry.Info("sync_started")

	if err := pgStore.EnsureSchema(ctx); err != nil {
		syncStatus = "failure"
		syncErrorMessage = err.Error()
		if errorsCounter != nil {
			telemetry.AddInt64Counter(ctx, errorsCounter, 1)
		}
		return fmt.Errorf("schema_ensure_failed: %w", err)
	}

	batchSize := int64(100)
	if envSize := os.Getenv("BATCH_SIZE"); envSize != "" {
		if val, err := strconv.ParseInt(envSize, 10, 64); err == nil && val > 0 {
			batchSize = val
		}
	}

	docs, err := mStore.FetchIngestedArticles(ctx, batchSize)
	if err != nil {
		syncStatus = "failure"
		syncErrorMessage = err.Error()
		if errorsCounter != nil {
			telemetry.AddInt64Counter(ctx, errorsCounter, 1)
		}
		return fmt.Errorf("fetch_from_mongo_failed: %w", err)
	}

	span.SetAttributes(telemetry.IntAttribute("db.documents.count", len(docs)))

	for _, doc := range docs {
		payloadJSON, _ := json.Marshal(doc.Payload)
		metaJSON, _ := json.Marshal(doc.Meta)

		err := pgStore.InsertReadingAnalytics(ctx, doc.ID, doc.Timestamp, doc.Source, doc.Type, payloadJSON, metaJSON)
		if err != nil {
			telemetry.Error("postgres_insert_failed", "id", doc.ID, "error", err)
			if errorsCounter != nil {
				telemetry.AddInt64Counter(ctx, errorsCounter, 1)
			}
			continue
		}

		if err := mStore.MarkArticleAsProcessed(ctx, doc.ID); err != nil {
			telemetry.Warn("mongo_mark_processed_failed", "id", doc.ID, "error", err)
			if errorsCounter != nil {
				telemetry.AddInt64Counter(ctx, errorsCounter, 1)
			}
		} else {
			processedCount++
		}
	}

	if processedCount > 0 && readingReady {
		telemetry.AddInt64Counter(ctx, processedCounter, int64(processedCount))
	}

	durationMs := time.Since(start).Milliseconds()
	if durationHist != nil {
		telemetry.RecordInt64Histogram(ctx, durationHist, durationMs)
	}

	telemetry.Info("sync_complete",
		"processed_count", processedCount,
		"duration", time.Since(start).String())

	return nil
}

// MongoStoreAPI defines the interface for MongoDB operations.
type MongoStoreAPI interface {
	FetchIngestedArticles(ctx context.Context, limit int64) ([]ReadingDocument, error)
	MarkArticleAsProcessed(ctx context.Context, id string) error
	Close(ctx context.Context) error
}

type MongoStoreWrapper struct {
	Wrapper *mongodb.MongoStore
}

func newMongoStore(store secrets.SecretStore) (MongoStoreAPI, error) {
	wrapper, err := mongodb.NewMongoStore(store)
	if err != nil {
		return nil, err
	}
	return &MongoStoreWrapper{Wrapper: wrapper}, nil
}

func (m *MongoStoreWrapper) FetchIngestedArticles(ctx context.Context, limit int64) ([]ReadingDocument, error) {
	var docs []ReadingDocument
	filter := map[string]any{"status": "ingested"}
	err := m.Wrapper.Find(ctx, "db.mongodb.fetch_ingested_articles", MongoDatabase, MongoCollection, filter, &docs, limit)
	if err != nil {
		return nil, err
	}
	return docs, nil
}

func (m *MongoStoreWrapper) MarkArticleAsProcessed(ctx context.Context, id string) error {
	update := map[string]any{"$set": map[string]any{"status": "processed"}}
	return m.Wrapper.UpdateByID(ctx, "db.mongodb.mark_article_processed", MongoDatabase, MongoCollection, id, update)
}

func (m *MongoStoreWrapper) Close(ctx context.Context) error {
	return m.Wrapper.Close(ctx)
}
