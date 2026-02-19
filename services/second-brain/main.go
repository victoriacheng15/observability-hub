package main

import (
	"context"
	"log/slog"
	"os"
	"sort"
	"sync"
	"time"

	"brain"
	"db/postgres"
	"env"
	"secrets"
	"telemetry"
)

var (
	brainOnce   sync.Once
	ready       bool
	syncTotal   telemetry.Int64Counter
	atomsGained telemetry.Int64Counter
)

func ensureMetrics() {
	brainOnce.Do(func() {
		meter := telemetry.GetMeter("second-brain")
		syncTotal, _ = telemetry.NewInt64Counter(meter, "second_brain.sync.total", "Total sync runs")
		atomsGained, _ = telemetry.NewInt64Counter(meter, "second_brain.atoms.ingested", "Total thoughts ingested")
		ready = true
	})
}

// BrainAPI defines the interface for interacting with GitHub journals
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

// App holds dependencies for the second-brain service
type App struct {
	SecretProviderFn func() (secrets.SecretStore, error)
	PostgresConnFn   func(driver string, store secrets.SecretStore) (*postgres.BrainStore, error)
	BrainAPI         BrainAPI
}

func main() {
	app := &App{
		SecretProviderFn: func() (secrets.SecretStore, error) {
			return secrets.NewBaoProvider()
		},
		PostgresConnFn: func(driver string, store secrets.SecretStore) (*postgres.BrainStore, error) {
			conn, err := postgres.ConnectPostgres(driver, store)
			if err != nil {
				return nil, err
			}
			return postgres.NewBrainStore(conn), nil
		},
		BrainAPI: &realBrainAPI{},
	}

	if err := app.Run(context.Background()); err != nil {
		slog.Error("second_brain_failed", "error", err)
		os.Exit(1)
	}
}

func (a *App) Run(ctx context.Context) error {
	// 1. Telemetry Init
	shutdownTracer, shutdownMeter, shutdownLogger, err := telemetry.Init(ctx, "second-brain")
	if err != nil {
		slog.Warn("otel_init_failed", "error", err)
	}
	defer func() {
		sCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownTracer != nil {
			_ = shutdownTracer(sCtx)
		}
		if shutdownMeter != nil {
			_ = shutdownMeter(sCtx)
		}
		if shutdownLogger != nil {
			_ = shutdownLogger(sCtx)
		}
	}()

	env.Load()
	ensureMetrics()

	repo := os.Getenv("JOURNAL_REPO")
	if repo == "" {
		return os.ErrNotExist
	}

	// 2. Initialize Secrets & DB
	store, err := a.SecretProviderFn()
	if err != nil {
		return err
	}
	defer store.Close()

	brainStore, err := a.PostgresConnFn("postgres", store)
	if err != nil {
		return err
	}

	return a.Sync(ctx, repo, brainStore)
}

func (a *App) Sync(ctx context.Context, repo string, brainStore *postgres.BrainStore) error {
	tracer := telemetry.GetTracer("second-brain")
	ctx, span := tracer.Start(ctx, "job.second_brain_sync")
	defer span.End()

	if ready {
		telemetry.AddInt64Counter(ctx, syncTotal, 1)
	}

	// 3. Get Latest Entry
	latestDate, err := brainStore.GetLatestEntryDate(ctx)
	if err != nil {
		return err
	}
	slog.Info("database_check_complete", "latest_entry", latestDate)

	// 4. Fetch delta
	_, fSpan := tracer.Start(ctx, "github.fetch_issues")
	allIssues, err := a.BrainAPI.FetchRecentJournals(repo)
	fSpan.End()
	if err != nil {
		return err
	}

	var newIssues []brain.GitHubIssue
	for _, iss := range allIssues {
		if iss.Title > latestDate {
			newIssues = append(newIssues, iss)
		}
	}

	if len(newIssues) == 0 {
		slog.Info("sync_skipped", "reason", "already_up_to_date")
		return nil
	}

	sort.Slice(newIssues, func(i, j int) bool { return newIssues[i].Title < newIssues[j].Title })

	// 5. Ingest
	totalAtoms := 0
	for _, iss := range newIssues {
		iCtx, iSpan := tracer.Start(ctx, "ingest.delta")
		iSpan.SetAttributes(
			telemetry.IntAttribute("github.issue_number", iss.Number),
			telemetry.StringAttribute("issue.title", iss.Title),
		)

		slog.Info("ingesting_issue", "number", iss.Number, "title", iss.Title)
		body, err := a.BrainAPI.FetchIssueBody(repo, iss.Number)
		if err != nil {
			slog.Error("fetch_body_failed", "issue", iss.Number, "error", err)
			iSpan.End()
			continue
		}

		atoms := brain.Atomize(iss.Title, body)
		for _, a := range atoms {
			if err := brainStore.InsertThought(iCtx, a.Date, a.Content, a.Category, a.Tags, a.ContextString, a.Checksum, a.TokenCount); err != nil {
				slog.Error("atom_insert_failed", "checksum", a.Checksum, "error", err)
				continue
			}
			if ready {
				telemetry.AddInt64Counter(iCtx, atomsGained, 1)
			}
			totalAtoms++
		}
		iSpan.SetAttributes(telemetry.IntAttribute("atoms.count", len(atoms)))
		iSpan.End()
	}

	slog.Info("sync_complete", "new_atoms", totalAtoms)
	return nil
}
