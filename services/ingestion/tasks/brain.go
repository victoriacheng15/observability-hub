package tasks

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"

	"brain"
	"db/postgres"
	"secrets"
	"telemetry"
)

// BrainTask implements the Task interface for syncing second brain data.
type BrainTask struct{}

var (
	brainOnce   sync.Once
	ready       bool
	syncTotal   telemetry.Int64Counter
	atomsGained telemetry.Int64Counter
	tokensTotal telemetry.Int64Counter
)

func ensureMetrics() {
	brainOnce.Do(func() {
		meterObj := telemetry.GetMeter("second.brain")
		syncTotal, _ = telemetry.NewInt64Counter(meterObj, "second.brain.sync.total", "Total sync runs")
		atomsGained, _ = telemetry.NewInt64Counter(meterObj, "second.brain.atoms.ingested", "Total thoughts ingested")
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
	ensureMetrics()

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

	if ready {
		telemetry.AddInt64Counter(ctx, syncTotal, 1)
	}

	latestDate, err := brainStore.GetLatestEntryDate(ctx)
	if err != nil {
		return err
	}
	telemetry.Info("database_check_complete", "latest_entry", latestDate)

	_, fSpan := tracer.Start(ctx, "github.fetch")
	allIssues, err := api.FetchRecentJournals(repo)
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
		telemetry.Info("sync_skipped", "reason", "already_up_to_date")
		return nil
	}

	sort.Slice(newIssues, func(i, j int) bool { return newIssues[i].Title < newIssues[j].Title })

	totalAtoms := 0
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
				continue
			}
			if ready {
				telemetry.AddInt64Counter(pCtx, atomsGained, 1)
				telemetry.AddInt64Counter(pCtx, tokensTotal, int64(a.TokenCount))
			}
			totalAtoms++
		}
		iSpan.SetAttributes(telemetry.IntAttribute("atoms.count", len(atoms)))
		iSpan.End()
	}

	telemetry.Info("sync_complete", "new_atoms", totalAtoms)
	return nil
}

// --- Store logic copied from services/second-brain/store.go ---

const (
	tableSecondBrain = "second_brain"
)

type BrainStore struct {
	Wrapper *postgres.PostgresWrapper
}

func NewBrainStore(w *postgres.PostgresWrapper) *BrainStore {
	return &BrainStore{Wrapper: w}
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

