package ingestion

import (
	"context"
	"testing"
	"time"

	"observability-hub/internal/db/postgres"
	"observability-hub/internal/worker/ingestion/brain"
	"observability-hub/internal/worker/store"
)

// --- Mocks ---

type mockBrainAPI struct {
	fetchRecentJournalsFn func(repo string) ([]brain.GitHubIssue, error)
	fetchIssueBodyFn      func(repo string, number int) (string, error)
}

func (m *mockBrainAPI) FetchRecentJournals(repo string) ([]brain.GitHubIssue, error) {
	return m.fetchRecentJournalsFn(repo)
}
func (m *mockBrainAPI) FetchIssueBody(repo string, number int) (string, error) {
	return m.fetchIssueBodyFn(repo, number)
}

type mockMongoAPI struct {
	fetchIngestedArticlesFn  func(ctx context.Context, limit int64) ([]store.ReadingDocument, error)
	markArticleAsProcessedFn func(ctx context.Context, id string) error
	closeFn                  func(ctx context.Context) error
}

func (m *mockMongoAPI) FetchIngestedArticles(ctx context.Context, limit int64) ([]store.ReadingDocument, error) {
	return m.fetchIngestedArticlesFn(ctx, limit)
}
func (m *mockMongoAPI) MarkArticleAsProcessed(ctx context.Context, id string) error {
	return m.markArticleAsProcessedFn(ctx, id)
}
func (m *mockMongoAPI) Close(ctx context.Context) error {
	if m.closeFn != nil {
		return m.closeFn(ctx)
	}
	return nil
}

type mockSecretStore struct{}

func (m *mockSecretStore) GetSecret(path, key, fallback string) string { return fallback }
func (m *mockSecretStore) Close() error                                { return nil }

// --- BrainTask Tests ---

func TestBrainTask_Sync(t *testing.T) {
	tests := []struct {
		name        string
		issues      []brain.GitHubIssue
		expectError bool
		setup       func(mdb *postgres.MockDB)
	}{
		{
			name:        "Success - New Issue",
			issues:      []brain.GitHubIssue{{Number: 1, Title: "2026-04-02"}},
			expectError: false,
			setup: func(mdb *postgres.MockDB) {
				rows := mdb.NewRows([]string{"max"}).AddRow("2026-04-01")
				mdb.ExpectQuery("SELECT COALESCE").WillReturnRows(rows)
				mdb.ExpectExec("INSERT INTO second_brain").WillReturnResult(mdb.NewResult(1, 1))
				mdb.ExpectExec("INSERT INTO second_brain_sync_history").WillReturnResult(mdb.NewResult(1, 1))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdb, cleanup := postgres.NewMockDB(t)
			defer cleanup()
			if tt.setup != nil {
				tt.setup(mdb)
			}

			workerStore := store.NewStore(mdb.Wrapper())
			task := &BrainTask{}
			api := &mockBrainAPI{
				fetchRecentJournalsFn: func(repo string) ([]brain.GitHubIssue, error) { return tt.issues, nil },
				fetchIssueBodyFn:      func(repo string, number int) (string, error) { return "## Thought\n- Content", nil },
			}

			err := task.Sync(context.Background(), "repo", workerStore, api)
			if (err != nil) != tt.expectError {
				t.Errorf("Sync() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// --- ReadingTask Tests ---

func TestReadingTask_Sync(t *testing.T) {
	tests := []struct {
		name        string
		docs        []store.ReadingDocument
		expectError bool
		setup       func(mdb *postgres.MockDB)
	}{
		{
			name:        "Success - New Docs",
			docs:        []store.ReadingDocument{{ID: "1", Source: "test", Type: "type", Timestamp: time.Now()}},
			expectError: false,
			setup: func(mdb *postgres.MockDB) {
				mdb.ExpectExec("INSERT INTO reading_analytics").WillReturnResult(mdb.NewResult(1, 1))
				mdb.ExpectExec("INSERT INTO reading_sync_history").WillReturnResult(mdb.NewResult(1, 1))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mdb, cleanup := postgres.NewMockDB(t)
			defer cleanup()
			if tt.setup != nil {
				tt.setup(mdb)
			}

			workerStore := store.NewStore(mdb.Wrapper())
			task := &ReadingTask{}
			mStore := &mockMongoAPI{
				fetchIngestedArticlesFn:  func(ctx context.Context, limit int64) ([]store.ReadingDocument, error) { return tt.docs, nil },
				markArticleAsProcessedFn: func(ctx context.Context, id string) error { return nil },
			}

			err := task.Sync(context.Background(), workerStore, mStore)
			if (err != nil) != tt.expectError {
				t.Errorf("Sync() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}
