package utils

import (
	"context"
	"encoding/json"
	"logger"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"db/mongodb"
	"db/postgres"
	"telemetry"
)

var readingTracer = telemetry.GetTracer("proxy/utils")
var readingMeter = telemetry.GetMeter("proxy/reading")

var (
	readingMetricsOnce     sync.Once
	readingMetricsReady    bool
	readingProcessedTotal  telemetry.Int64Counter
	readingErrorsTotal     telemetry.Int64Counter
	readingBatchSizeHist   telemetry.Int64Histogram
	readingRequestDuration telemetry.Int64Histogram
)

type MongoStoreAPI interface {
	FetchIngestedArticles(ctx context.Context, limit int64) ([]mongodb.ReadingDocument, error)
	MarkArticleAsProcessed(ctx context.Context, id string) error
}

type ReadingService struct {
	Store      *postgres.ReadingStore
	MongoStore MongoStoreAPI
}

func ensureReadingMetrics() {
	readingMetricsOnce.Do(func() {
		var err error
		readingProcessedTotal, err = telemetry.NewInt64Counter(
			readingMeter,
			"proxy.sync.reading.processed.total",
			"Total reading documents processed",
		)
		if err != nil {
			logger.Warn("reading_metric_init_failed", "metric", "proxy.sync.reading.processed.total", "error", err)
			return
		}

		readingErrorsTotal, err = telemetry.NewInt64Counter(
			readingMeter,
			"proxy.sync.reading.errors.total",
			"Total reading sync errors",
		)
		if err != nil {
			logger.Warn("reading_metric_init_failed", "metric", "proxy.sync.reading.errors.total", "error", err)
			return
		}

		readingBatchSizeHist, err = telemetry.NewInt64Histogram(
			readingMeter,
			"proxy.sync.reading.batch.size",
			"Reading sync batch size",
			"count",
		)
		if err != nil {
			logger.Warn("reading_metric_init_failed", "metric", "proxy.sync.reading.batch.size", "error", err)
			return
		}

		readingRequestDuration, err = telemetry.NewInt64Histogram(
			readingMeter,
			"proxy.sync.reading.duration.ms",
			"Reading sync request duration in milliseconds",
			"ms",
		)
		if err != nil {
			logger.Warn("reading_metric_init_failed", "metric", "proxy.sync.reading.duration.ms", "error", err)
			return
		}

		readingMetricsReady = true
	})
}

func (s *ReadingService) SyncReadingHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ensureReadingMetrics()

	ctx, handlerSpan := readingTracer.Start(r.Context(), "handler.sync_reading")
	defer handlerSpan.End()

	metricAttrs := []telemetry.Attribute{
		telemetry.StringAttribute("handler", "sync_reading"),
	}
	defer func() {
		if readingMetricsReady {
			durationMs := time.Since(start).Milliseconds()
			telemetry.RecordInt64Histogram(ctx, readingRequestDuration, durationMs, metricAttrs...)
		}
	}()

	startTime := time.Now().UTC()
	syncStatus := "success"
	var syncErrorMessage string
	processedCount := 0

	defer func() {
		if err := s.Store.RecordSyncHistory(ctx, startTime, time.Now().UTC(), syncStatus, processedCount, syncErrorMessage); err != nil {
			logger.Error("sync_reading_history_record_failed", "error", err)
		}
	}()

	logger.Info("sync_reading_started")

	if err := s.Store.EnsureSchema(ctx); err != nil {
		if readingMetricsReady {
			telemetry.AddInt64Counter(ctx, readingErrorsTotal, 1, metricAttrs...)
		}
		handlerSpan.SetStatus(telemetry.CodeError, "ensure_schema_failed")
		handlerSpan.SetAttributes(
			telemetry.BoolAttribute("error", true),
			telemetry.StringAttribute("error.message", err.Error()),
		)
		logger.Error("sync_reading_schema_failed", "error", err)
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
		if readingMetricsReady {
			telemetry.AddInt64Counter(ctx, readingErrorsTotal, 1, metricAttrs...)
		}
		handlerSpan.SetStatus(telemetry.CodeError, "fetch_failed")
		handlerSpan.SetAttributes(
			telemetry.BoolAttribute("error", true),
			telemetry.StringAttribute("error.message", err.Error()),
		)
		logger.Error("sync_reading_fetch_failed", "error", err)
		http.Error(w, "Failed to query Mongo", 500)
		syncStatus = "failure"
		syncErrorMessage = err.Error()
		return
	}
	fetchSpan.SetAttributes(telemetry.IntAttribute("batch.fetched", len(docs)))
	fetchSpan.End()
	if readingMetricsReady {
		telemetry.RecordInt64Histogram(ctx, readingBatchSizeHist, int64(len(docs)), metricAttrs...)
	}

	for _, doc := range docs {
		payloadJSON, _ := json.Marshal(doc.Payload)
		metaJSON, _ := json.Marshal(doc.Meta)

		insertCtx, insertSpan := readingTracer.Start(ctx, "sync.insert_to_postgres")
		err := s.Store.InsertReadingAnalytics(insertCtx, doc.ID, doc.Timestamp, doc.Source, doc.Type, payloadJSON, metaJSON)
		if err != nil {
			insertSpan.RecordError(err)
			insertSpan.SetStatus(telemetry.CodeError, "failed to insert")
			insertSpan.End()
			if readingMetricsReady {
				telemetry.AddInt64Counter(ctx, readingErrorsTotal, 1, metricAttrs...)
			}
			logger.Error("sync_reading_insert_failed", "id", doc.ID, "error", err)
			continue
		}
		insertSpan.End()

		markCtx, markSpan := readingTracer.Start(ctx, "sync.mark_processed")
		if err := s.MongoStore.MarkArticleAsProcessed(markCtx, doc.ID); err != nil {
			markSpan.RecordError(err)
			markSpan.SetStatus(telemetry.CodeError, "failed to mark processed")
			markSpan.End()
			if readingMetricsReady {
				telemetry.AddInt64Counter(ctx, readingErrorsTotal, 1, metricAttrs...)
			}
			logger.Warn("sync_reading_mark_failed", "id", doc.ID, "error", err)
		} else {
			markSpan.End()
			processedCount++
		}
	}

	if readingMetricsReady && processedCount > 0 {
		telemetry.AddInt64Counter(ctx, readingProcessedTotal, int64(processedCount), metricAttrs...)
	}

	res := map[string]interface{}{
		"service":         "reading-sync",
		"status":          syncStatus,
		"processed_count": processedCount,
		"timestamp":       time.Now().UTC(),
	}

	logger.Info("sync_reading_batch_complete", "details", res)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
