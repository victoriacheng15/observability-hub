package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"db/mongodb"
	"db/postgres"
	"env"
	"secrets"
	"telemetry"
)

type MongoStoreAPI interface {
	FetchIngestedArticles(ctx context.Context, limit int64) ([]mongodb.ReadingDocument, error)
	MarkArticleAsProcessed(ctx context.Context, id string) error
	Close(ctx context.Context) error
}

// App holds dependencies for the reading-sync service
type App struct {
	SecretProviderFn func() (secrets.SecretStore, error)
	PostgresConnFn   func(driver string, store secrets.SecretStore) (*postgres.ReadingStore, error)
	MongoConnFn      func(store secrets.SecretStore) (MongoStoreAPI, error)
}

func main() {
	app := &App{
		SecretProviderFn: func() (secrets.SecretStore, error) {
			return secrets.NewBaoProvider()
		},
		PostgresConnFn: func(driver string, store secrets.SecretStore) (*postgres.ReadingStore, error) {
			conn, err := postgres.ConnectPostgres(driver, store)
			if err != nil {
				return nil, err
			}
			return postgres.NewReadingStore(conn), nil
		},
		MongoConnFn: func(store secrets.SecretStore) (MongoStoreAPI, error) {
			return mongodb.NewMongoStore(store)
		},
	}

	if err := app.Run(context.Background()); err != nil {
		telemetry.Error("reading_sync_failed", "error", err)
		os.Exit(1)
	}
}

func (a *App) Run(ctx context.Context) error {
	env.Load()

	// 1. Telemetry
	shutdownTracer, shutdownMeter, shutdownLogger, err := telemetry.Init(ctx, "reading.sync")
	if err != nil {
		telemetry.Warn("otel_init_failed, continuing without full observability", "error", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownTracer != nil {
			if err := shutdownTracer(shutdownCtx); err != nil {
				telemetry.Error("otel_shutdown_failed", "component", "tracer", "error", err)
			}
		}
		if shutdownMeter != nil {
			if err := shutdownMeter(shutdownCtx); err != nil {
				telemetry.Error("otel_shutdown_failed", "component", "meter", "error", err)
			}
		}
		if shutdownLogger != nil {
			if err := shutdownLogger(shutdownCtx); err != nil {
				telemetry.Error("otel_shutdown_failed", "component", "logger", "error", err)
			}
		}
	}()

	// 2. Secrets
	secretStore, err := a.SecretProviderFn()
	if err != nil {
		return fmt.Errorf("secret_provider_init_failed: %w", err)
	}
	defer secretStore.Close()

	// 3. Postgres
	readingStore, err := a.PostgresConnFn("postgres", secretStore)
	if err != nil {
		return fmt.Errorf("postgres_connection_failed: %w", err)
	}
	defer readingStore.DB.Close()

	// 4. Mongo
	mongoStore, err := a.MongoConnFn(secretStore)
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

	// 5. Execution
	return a.Sync(ctx, readingStore, mongoStore)
}

func (a *App) Sync(ctx context.Context, pgStore *postgres.ReadingStore, mStore MongoStoreAPI) error {
	tracer := telemetry.GetTracer("reading.sync")
	meter := telemetry.GetMeter("reading.sync")

	processedCounter, _ := telemetry.NewInt64Counter(meter, "reading.sync.processed.total", "Total documents processed")
	errorsCounter, _ := telemetry.NewInt64Counter(meter, "reading.sync.errors.total", "Total sync errors")
	durationHist, _ := telemetry.NewInt64Histogram(meter, "reading.sync.duration.ms", "Sync duration in milliseconds", "ms")
	batchSizeHist, _ := telemetry.NewInt64Histogram(meter, "reading.sync.batch.size", "Number of documents processed in a batch", "count")

	start := time.Now()
	ctx, span := tracer.Start(ctx, "job.reading_sync")
	defer span.End()

	startTime := time.Now().UTC()
	syncStatus := "success"
	var syncErrorMessage string
	processedCount := 0

	defer func() {
		if err := pgStore.RecordSyncHistory(ctx, startTime, time.Now().UTC(), syncStatus, processedCount, syncErrorMessage); err != nil {
			telemetry.Error("failed_to_record_sync_history", "error", err)
		}
	}()

	telemetry.Info("sync_started")

	// Ensure Schema
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

	// Fetch
	docs, err := mStore.FetchIngestedArticles(ctx, batchSize)
	if err != nil {
		syncStatus = "failure"
		syncErrorMessage = err.Error()
		if errorsCounter != nil {
			telemetry.AddInt64Counter(ctx, errorsCounter, 1)
		}
		return fmt.Errorf("fetch_from_mongo_failed: %w", err)
	}

	if batchSizeHist != nil {
		telemetry.RecordInt64Histogram(ctx, batchSizeHist, int64(len(docs)))
	}

	for _, doc := range docs {
		payloadJSON, _ := json.Marshal(doc.Payload)
		metaJSON, _ := json.Marshal(doc.Meta)

		// Insert to Postgres
		err := pgStore.InsertReadingAnalytics(ctx, doc.ID, doc.Timestamp, doc.Source, doc.Type, payloadJSON, metaJSON)
		if err != nil {
			telemetry.Error("postgres_insert_failed", "id", doc.ID, "error", err)
			if errorsCounter != nil {
				telemetry.AddInt64Counter(ctx, errorsCounter, 1)
			}
			continue
		}

		// Mark as Processed in Mongo
		if err := mStore.MarkArticleAsProcessed(ctx, doc.ID); err != nil {
			telemetry.Warn("mongo_mark_processed_failed", "id", doc.ID, "error", err)
			if errorsCounter != nil {
				telemetry.AddInt64Counter(ctx, errorsCounter, 1)
			}
		} else {
			processedCount++
		}
	}

	if processedCount > 0 && processedCounter != nil {
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
