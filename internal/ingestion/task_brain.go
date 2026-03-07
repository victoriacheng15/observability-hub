package ingestion

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"observability-hub/internal/db/postgres"
	"observability-hub/internal/ingestion/brain"
	"observability-hub/internal/secrets"
	"observability-hub/internal/telemetry"
)

// BrainTask implements the Task interface for syncing second brain data.
type BrainTask struct{}

var (
	brainOnce sync.Once
	ready     bool
	// Global metrics initialized once
	processedItemsCounter telemetry.Int64Counter
	tokensTotal           telemetry.Int64Counter
)

func ensureGlobalMetrics() {
	brainOnce.Do(func() {
		meterObj := telemetry.GetMeter("second.brain")
		processedItemsCounter, _ = telemetry.NewInt64Counter(meterObj, "second.brain.sync.processed.total", "Total thoughts ingested")
		tokensTotal, _ = telemetry.NewInt64Counter(meterObj, "second.brain.token.count.total", "Total tokens processed")
		ready = true
	})
}

// Name returns the name of the task.
func (t *BrainTask) Name() string {
	return "brain"
}

var newBrainAPI = func() brain.BrainAPI {
	return brain.NewBrainAPI()
}

// Run executes the brain sync task.
func (t *BrainTask) Run(ctx context.Context, db *postgres.PostgresWrapper, secretStore secrets.SecretStore) error {
	ensureGlobalMetrics()

	repo := os.Getenv("JOURNAL_REPO")
	if repo == "" {
		return fmt.Errorf("JOURNAL_REPO environment variable not set")
	}

	brainStore := NewBrainStore(db)
	api := newBrainAPI()

	return t.Sync(ctx, repo, brainStore, api)
}

// Sync performs the data synchronization from GitHub to PostgreSQL.
func (t *BrainTask) Sync(ctx context.Context, repo string, brainStore *BrainStore, api brain.BrainAPI) error {
	tracer := telemetry.GetTracer("second.brain")
	meter := telemetry.GetMeter("second.brain")

	// Task-specific metrics
	syncTotal, _ := telemetry.NewInt64Counter(meter, "second.brain.sync.total", "Total sync runs")
	errorsCounter, _ := telemetry.NewInt64Counter(meter, "second.brain.sync.errors.total", "Total sync errors")
	durationHist, _ := telemetry.NewInt64Histogram(meter, "second.brain.sync.duration.ms", "Sync duration in milliseconds", "ms")

	lastSuccessTime := time.Now()
	_, _ = telemetry.NewInt64ObservableGauge(meter, "second.brain.sync.lag.seconds", "Time since last successful sync", func(ctx context.Context, obs telemetry.Int64Observer) error {
		obs.Observe(int64(time.Since(lastSuccessTime).Seconds()))
		return nil
	})

	if syncTotal != nil {
		telemetry.AddInt64Counter(ctx, syncTotal, 1)
	}

	start := time.Now()
	ctx, span := tracer.Start(ctx, "job.second_brain_sync")
	defer span.End()

	startTime := time.Now().UTC()
	syncStatus := "success"
	var syncErrorMessage string
	processedCount := 0

	defer func() {
		if syncStatus == "success" {
			lastSuccessTime = time.Now()
		}
		if err := brainStore.RecordSyncHistory(ctx, startTime, time.Now().UTC(), syncStatus, processedCount, syncErrorMessage); err != nil {
			telemetry.Error("failed_to_record_brain_sync_history", "error", err)
		}
	}()

	telemetry.Info("sync_started")

	// 1. Ensure Schema
	if err := brainStore.EnsureSchema(ctx); err != nil {
		syncStatus = "failure"
		syncErrorMessage = err.Error()
		if errorsCounter != nil {
			telemetry.AddInt64Counter(ctx, errorsCounter, 1)
		}
		return fmt.Errorf("schema_ensure_failed: %w", err)
	}

	// 2. Get Latest Entry Date
	latestDate, err := brainStore.GetLatestEntryDate(ctx)
	if err != nil {
		syncStatus = "failure"
		syncErrorMessage = err.Error()
		if errorsCounter != nil {
			telemetry.AddInt64Counter(ctx, errorsCounter, 1)
		}
		return err
	}
	telemetry.Info("database_check_complete", "latest_entry", latestDate)

	// 3. Fetch Recent Journals
	_, fSpan := tracer.Start(ctx, "github.fetch")
	allIssues, err := api.FetchRecentJournals(repo)
	fSpan.End()
	if err != nil {
		syncStatus = "failure"
		syncErrorMessage = err.Error()
		if errorsCounter != nil {
			telemetry.AddInt64Counter(ctx, errorsCounter, 1)
		}
		return err
	}

	var newIssues []brain.GitHubIssue
	for _, iss := range allIssues {
		if iss.Title > latestDate {
			newIssues = append(newIssues, iss)
		}
	}

	if len(newIssues) == 0 {
		telemetry.Info("sync_skipped", "reason", "already_up_to_date")
		return nil
	}

	sort.Slice(newIssues, func(i, j int) bool { return newIssues[i].Title < newIssues[j].Title })

	for _, iss := range newIssues {
		iCtx, iSpan := tracer.Start(ctx, "ingest.delta")
		iSpan.SetAttributes(
			telemetry.IntAttribute("github.issue_number", iss.Number),
			telemetry.StringAttribute("issue.title", iss.Title),
		)

		telemetry.Info("ingesting_issue", "number", iss.Number, "title", iss.Title)
		body, err := api.FetchIssueBody(repo, iss.Number)
		if err != nil {
			telemetry.Error("fetch_body_failed", "issue", iss.Number, "error", err)
			iSpan.End()
			continue
		}

		pCtx, pSpan := tracer.Start(iCtx, "parse.markdown.duration")
		atoms := brain.Atomize(iss.Title, body)
		pSpan.End()

		for _, a := range atoms {
			if err := brainStore.InsertThought(pCtx, a.Date, a.Content, a.Category, a.Tags, a.ContextString, a.Checksum, a.TokenCount); err != nil {
				telemetry.Error("atom_insert_failed", "checksum", a.Checksum, "error", err)
				if errorsCounter != nil {
					telemetry.AddInt64Counter(pCtx, errorsCounter, 1)
				}
				continue
			}
			if ready {
				telemetry.AddInt64Counter(pCtx, processedItemsCounter, 1)
				telemetry.AddInt64Counter(pCtx, tokensTotal, int64(a.TokenCount))
			}
			processedCount++
		}
		iSpan.SetAttributes(telemetry.IntAttribute("atoms.count", len(atoms)))
		iSpan.End()
	}

	durationMs := time.Since(start).Milliseconds()
	if durationHist != nil {
		telemetry.RecordInt64Histogram(ctx, durationHist, durationMs)
	}

	telemetry.Info("sync_complete", "new_atoms", processedCount)
	return nil
}
