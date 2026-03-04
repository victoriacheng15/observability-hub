package tasks

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"observability-hub/internal/brain"
	"observability-hub/internal/db/postgres"
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

// Run executes the brain sync task.
func (t *BrainTask) Run(ctx context.Context, db *postgres.PostgresWrapper, secretStore secrets.SecretStore) error {
	ensureGlobalMetrics()

	repo := os.Getenv("JOURNAL_REPO")
	if repo == "" {
		return fmt.Errorf("JOURNAL_REPO environment variable not set")
	}

	brainStore := NewBrainStore(db)
	api := &realBrainAPI{}

	return t.Sync(ctx, repo, brainStore, api)
}

// Sync performs the data synchronization from GitHub to PostgreSQL.
func (t *BrainTask) Sync(ctx context.Context, repo string, brainStore *BrainStore, api BrainAPI) error {
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

// --- Store logic ---

const (
	tableSecondBrain = "second_brain"
	tableBrainSync   = "second_brain_sync_history"
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
		)`, tableSecondBrain)

	_, err := s.Wrapper.Exec(ctx, "db.postgres.ensure_second_brain", queryBrain)
	if err != nil {
		return fmt.Errorf("failed to ensure %s table: %w", tableSecondBrain, err)
	}

	queryHistory := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id SERIAL PRIMARY KEY,
			start_time TIMESTAMPTZ NOT NULL,
			end_time TIMESTAMPTZ NOT NULL,
			status TEXT NOT NULL,
			processed_count INTEGER NOT NULL,
			error_message TEXT,
			created_at TIMESTAMPTZ DEFAULT NOW()
		)`, tableBrainSync)

	_, err = s.Wrapper.Exec(ctx, "db.postgres.ensure_brain_sync_history", queryHistory)
	if err != nil {
		return fmt.Errorf("failed to ensure %s table: %w", tableBrainSync, err)
	}

	return nil
}

func (s *BrainStore) RecordSyncHistory(ctx context.Context, startTime, endTime time.Time, status string, processedCount int, errorMessage string) error {
	query := fmt.Sprintf(`INSERT INTO %s (start_time, end_time, status, processed_count, error_message)
		VALUES ($1, $2, $3, $4, $5)`, tableBrainSync)

	_, err := s.Wrapper.Exec(ctx, "db.postgres.record_brain_sync_history", query, startTime, endTime, status, processedCount, errorMessage)
	return err
}

func (s *BrainStore) GetLatestEntryDate(ctx context.Context) (string, error) {
	var latestDate string
	query := fmt.Sprintf("SELECT COALESCE(MAX(entry_date)::text, '1970-01-01') FROM %s", tableSecondBrain)
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
		ON CONFLICT (checksum) DO NOTHING`, tableSecondBrain)

	_, err := s.Wrapper.Exec(ctx, "db.postgres.insert_thought", query,
		date, content, category, s.Wrapper.Array(tags), contextString, checksum, tokenCount)
	return err
}

// --- GitHub API implementation ---

type BrainAPI interface {
	FetchRecentJournals(repo string) ([]brain.GitHubIssue, error)
	FetchIssueBody(repo string, number int) (string, error)
}

type realBrainAPI struct{}

func (r *realBrainAPI) FetchRecentJournals(repo string) ([]brain.GitHubIssue, error) {
	return brain.FetchRecentJournals(repo)
}
func (r *realBrainAPI) FetchIssueBody(repo string, number int) (string, error) {
	return brain.FetchIssueBody(repo, number)
}
