package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"observability-hub/internal/db/mongodb"
	"observability-hub/internal/secrets"
	"observability-hub/internal/telemetry"
	"observability-hub/internal/worker/ingestion/brain"
	"observability-hub/internal/worker/store"
)

// Task defines the interface for a background ingestion task.
type Task interface {
	Name() string
	Run(ctx context.Context, s *store.Store, secretStore secrets.SecretStore) error
}

// --- Brain Task Implementation ---

type BrainTask struct{}

var (
	brainOnce             sync.Once
	brainReady            bool
	processedItemsCounter telemetry.Int64Counter
	tokensTotal           telemetry.Int64Counter
)

func ensureGlobalMetrics() {
	brainOnce.Do(func() {
		meterObj := telemetry.GetMeter("worker.ingestion")
		processedItemsCounter, _ = telemetry.NewInt64Counter(meterObj, "worker.brain.sync.processed.total", "Total thoughts ingested")
		tokensTotal, _ = telemetry.NewInt64Counter(meterObj, "worker.brain.token.count.total", "Total tokens processed")
		brainReady = true
	})
}

func (t *BrainTask) Name() string { return "brain" }

var newBrainAPI = func() brain.BrainAPI { return brain.NewBrainAPI() }

func (t *BrainTask) Run(ctx context.Context, s *store.Store, secretStore secrets.SecretStore) error {
	ensureGlobalMetrics()
	repo := os.Getenv("JOURNAL_REPO")
	if repo == "" {
		return fmt.Errorf("JOURNAL_REPO not set")
	}
	return t.Sync(ctx, repo, s, newBrainAPI())
}

func (t *BrainTask) Sync(ctx context.Context, repo string, s *store.Store, api brain.BrainAPI) error {
	tracer := telemetry.GetTracer("worker.ingestion")
	meter := telemetry.GetMeter("worker.ingestion")
	syncTotal, _ := telemetry.NewInt64Counter(meter, "worker.brain.sync.total", "Total sync runs")
	errorsCounter, _ := telemetry.NewInt64Counter(meter, "worker.brain.sync.errors.total", "Total sync errors")
	durationHist, _ := telemetry.NewInt64Histogram(meter, "worker.brain.sync.duration", "Sync duration", "ms")

	if syncTotal != nil {
		telemetry.AddInt64Counter(ctx, syncTotal, 1)
	}
	start := time.Now()
	ctx, span := tracer.Start(ctx, "job.second_brain_sync")
	defer span.End()

	startTime := time.Now().UTC()
	syncStatus := "success"
	var syncErr string
	processedCount := 0

	defer func() {
		if err := s.RecordSyncHistory(ctx, store.TableBrainSync, startTime, time.Now().UTC(), syncStatus, processedCount, syncErr); err != nil {
			telemetry.Error("failed_to_record_brain_sync_history", "error", err)
		}
	}()

	latestDate, err := s.GetLatestEntryDate(ctx)
	if err != nil {
		syncStatus = "failure"
		syncErr = err.Error()
		return err
	}

	allIssues, err := api.FetchRecentJournals(repo)
	if err != nil {
		syncStatus = "failure"
		syncErr = err.Error()
		return err
	}

	var newIssues []brain.GitHubIssue
	for _, iss := range allIssues {
		if iss.Title > latestDate {
			newIssues = append(newIssues, iss)
		}
	}
	if len(newIssues) == 0 {
		return nil
	}

	sort.Slice(newIssues, func(i, j int) bool { return newIssues[i].Title < newIssues[j].Title })

	for _, iss := range newIssues {
		body, err := api.FetchIssueBody(repo, iss.Number)
		if err != nil {
			continue
		}
		atoms := brain.Atomize(iss.Title, body)
		for _, a := range atoms {
			if err := s.InsertThought(ctx, a.Date, a.Content, a.Category, a.Tags, a.ContextString, a.Checksum, a.TokenCount); err != nil {
				if errorsCounter != nil {
					telemetry.AddInt64Counter(ctx, errorsCounter, 1)
				}
				continue
			}
			if brainReady {
				telemetry.AddInt64Counter(ctx, processedItemsCounter, 1)
				telemetry.AddInt64Counter(ctx, tokensTotal, int64(a.TokenCount))
			}
			processedCount++
		}
	}
	if durationHist != nil {
		telemetry.RecordInt64Histogram(ctx, durationHist, time.Since(start).Milliseconds())
	}
	return nil
}

// --- Reading Task Implementation ---

type ReadingTask struct{}

var (
	readingOnce             sync.Once
	readingReady            bool
	readingProcessedCounter telemetry.Int64Counter
)

func ensureReadingGlobalMetrics() {
	readingOnce.Do(func() {
		meterObj := telemetry.GetMeter("worker.ingestion")
		readingProcessedCounter, _ = telemetry.NewInt64Counter(meterObj, "worker.reading.sync.processed.total", "Total documents processed")
		readingReady = true
	})
}

func (t *ReadingTask) Name() string { return "reading" }

func (t *ReadingTask) Run(ctx context.Context, s *store.Store, secretStore secrets.SecretStore) error {
	ensureReadingGlobalMetrics()
	mStore, err := newMongoStore(secretStore)
	if err != nil {
		return err
	}
	defer mStore.Close(ctx)
	return t.Sync(ctx, s, mStore)
}

func (t *ReadingTask) Sync(ctx context.Context, s *store.Store, mStore MongoStoreAPI) error {
	tracer := telemetry.GetTracer("worker.ingestion")
	meter := telemetry.GetMeter("worker.ingestion")
	syncTotal, _ := telemetry.NewInt64Counter(meter, "worker.reading.sync.total", "Total sync runs")
	errorsCounter, _ := telemetry.NewInt64Counter(meter, "worker.reading.sync.errors.total", "Total sync errors")
	durationHist, _ := telemetry.NewInt64Histogram(meter, "worker.reading.sync.duration", "Sync duration", "ms")

	if syncTotal != nil {
		telemetry.AddInt64Counter(ctx, syncTotal, 1)
	}
	start := time.Now()
	ctx, span := tracer.Start(ctx, "job.reading_sync")
	defer span.End()

	startTime := time.Now().UTC()
	syncStatus := "success"
	var syncErr string
	processedCount := 0

	defer func() {
		if err := s.RecordSyncHistory(ctx, store.TableReadingSync, startTime, time.Now().UTC(), syncStatus, processedCount, syncErr); err != nil {
			telemetry.Error("failed_to_record_reading_sync_history", "error", err)
		}
	}()

	docs, err := mStore.FetchIngestedArticles(ctx, 100)
	if err != nil {
		syncStatus = "failure"
		syncErr = err.Error()
		return err
	}

	for _, doc := range docs {
		pJSON, _ := json.Marshal(doc.Payload)
		mJSON, _ := json.Marshal(doc.Meta)
		if err := s.InsertReadingAnalytics(ctx, doc.ID, doc.Timestamp, doc.Source, doc.Type, pJSON, mJSON); err != nil {
			if errorsCounter != nil {
				telemetry.AddInt64Counter(ctx, errorsCounter, 1)
			}
			continue
		}
		if err := mStore.MarkArticleAsProcessed(ctx, doc.ID); err == nil {
			processedCount++
		}
	}

	if processedCount > 0 && readingReady {
		telemetry.AddInt64Counter(ctx, readingProcessedCounter, int64(processedCount))
	}
	if durationHist != nil {
		telemetry.RecordInt64Histogram(ctx, durationHist, time.Since(start).Milliseconds())
	}
	return nil
}

type MongoStoreAPI interface {
	FetchIngestedArticles(ctx context.Context, limit int64) ([]store.ReadingDocument, error)
	MarkArticleAsProcessed(ctx context.Context, id string) error
	Close(ctx context.Context) error
}

type MongoStoreWrapper struct{ Wrapper *mongodb.MongoStore }

var newMongoStore = func(store secrets.SecretStore) (MongoStoreAPI, error) {
	wrapper, err := mongodb.NewMongoStore(store)
	if err != nil {
		return nil, err
	}
	return &MongoStoreWrapper{Wrapper: wrapper}, nil
}

func (m *MongoStoreWrapper) FetchIngestedArticles(ctx context.Context, limit int64) ([]store.ReadingDocument, error) {
	var docs []store.ReadingDocument
	err := m.Wrapper.Find(ctx, "db.mongodb.fetch_ingested_articles", "reading-analytics", "articles", map[string]any{"status": "ingested"}, &docs, limit)
	return docs, err
}

func (m *MongoStoreWrapper) MarkArticleAsProcessed(ctx context.Context, id string) error {
	return m.Wrapper.UpdateByID(ctx, "db.mongodb.mark_article_processed", "reading-analytics", "articles", id, map[string]any{"$set": map[string]any{"status": "processed"}})
}

func (m *MongoStoreWrapper) Close(ctx context.Context) error { return m.Wrapper.Close(ctx) }
