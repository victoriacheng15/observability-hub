package utils

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"db/mongodb"
	"db/postgres"
	"telemetry"
)

var readingTracer = telemetry.GetTracer("proxy/utils")

type MongoStoreAPI interface {
	FetchIngestedArticles(ctx context.Context, limit int64) ([]mongodb.ReadingDocument, error)
	MarkArticleAsProcessed(ctx context.Context, id string) error
}

type ReadingService struct {
	Store      *postgres.ReadingStore
	MongoStore MongoStoreAPI
}

func (s *ReadingService) SyncReadingHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startTime := time.Now().UTC()
	syncStatus := "success"
	var syncErrorMessage string
	processedCount := 0

	defer func() {
		if err := s.Store.RecordSyncHistory(ctx, startTime, time.Now().UTC(), syncStatus, processedCount, syncErrorMessage); err != nil {
			slog.Error("ETL_ERROR: Failed to record sync history", "error", err)
		}
	}()

	if err := s.Store.EnsureSchema(ctx); err != nil {
		slog.Error("ETL_ERROR: Failed to ensure database schema", "error", err)
		http.Error(w, "Failed to ensure database schema", 500)
		syncStatus = "failure"
		syncErrorMessage = err.Error()
		return
	}

	batchSize := int64(100)
	if envSize := os.Getenv("BATCH_SIZE"); envSize != "" {
		if val, err := strconv.ParseInt(envSize, 10, 64); err == nil && val > 0 {
			batchSize = val
		}
	}

	ctx, fetchSpan := readingTracer.Start(ctx, "sync.fetch_from_mongo")
	docs, err := s.MongoStore.FetchIngestedArticles(ctx, batchSize)
	if err != nil {
		fetchSpan.RecordError(err)
		fetchSpan.SetStatus(telemetry.CodeError, "failed to query mongo")
		fetchSpan.End()
		slog.Error("ETL_ERROR: Failed to query Mongo", "error", err)
		http.Error(w, "Failed to query Mongo", 500)
		syncStatus = "failure"
		syncErrorMessage = err.Error()
		return
	}
	fetchSpan.SetAttributes(telemetry.IntAttribute("batch.fetched", len(docs)))
	fetchSpan.End()

	for _, doc := range docs {
		payloadJSON, _ := json.Marshal(doc.Payload)
		metaJSON, _ := json.Marshal(doc.Meta)

		insertCtx, insertSpan := readingTracer.Start(ctx, "sync.insert_to_postgres")
		err := s.Store.InsertReadingAnalytics(insertCtx, doc.ID, doc.Timestamp, doc.Source, doc.Type, payloadJSON, metaJSON)
		if err != nil {
			insertSpan.RecordError(err)
			insertSpan.SetStatus(telemetry.CodeError, "failed to insert")
			insertSpan.End()
			slog.Error("ETL_ERROR: Failed to insert into Postgres", "id", doc.ID, "error", err)
			continue
		}
		insertSpan.End()

		markCtx, markSpan := readingTracer.Start(ctx, "sync.mark_processed")
		if err := s.MongoStore.MarkArticleAsProcessed(markCtx, doc.ID); err != nil {
			markSpan.RecordError(err)
			markSpan.SetStatus(telemetry.CodeError, "failed to mark processed")
			markSpan.End()
			slog.Warn("ETL_WARN: Failed to update Mongo status", "id", doc.ID, "error", err)
		} else {
			markSpan.End()
			processedCount++
		}
	}

	res := map[string]interface{}{
		"service":         "reading-sync",
		"status":          syncStatus,
		"processed_count": processedCount,
		"timestamp":       time.Now().UTC(),
	}

	slog.Info("ETL_SUCCESS: Processed batch", "details", res)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
