package ingestion

import (
	"context"
	"errors"
	"os"
	"testing"

	"observability-hub/internal/db/postgres"
	"observability-hub/internal/ingestion/brain"

	"github.com/DATA-DOG/go-sqlmock"
)

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

func TestBrainTask_Name(t *testing.T) {
	task := &BrainTask{}
	if task.Name() != "brain" {
		t.Errorf("Expected brain, got %s", task.Name())
	}
}

func TestBrainTask_Run(t *testing.T) {
	tests := []struct {
		name    string
		envRepo string
		wantErr bool
	}{
		{
			name:    "Missing Repo",
			envRepo: "",
			wantErr: true,
		},
		{
			name:    "Success",
			envRepo: "test/repo",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envRepo != "" {
				t.Setenv("JOURNAL_REPO", tt.envRepo)
			} else {
				os.Unsetenv("JOURNAL_REPO")
			}

			db, mock, _ := sqlmock.New()
			defer db.Close()
			wrapper := &postgres.PostgresWrapper{DB: db}

			if !tt.wantErr {
				// Expectations for Sync
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow("2023-01-01"))
				mock.ExpectExec("INSERT INTO second_brain_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "success", 0, "").
					WillReturnResult(sqlmock.NewResult(1, 1))

				oldNewBrainAPI := newBrainAPI
				defer func() { newBrainAPI = oldNewBrainAPI }()
				newBrainAPI = func() brain.BrainAPI {
					return &mockBrainAPI{
						fetchRecentJournalsFn: func(repo string) ([]brain.GitHubIssue, error) {
							return nil, nil
						},
					}
				}
			}

			task := &BrainTask{}
			err := task.Run(context.Background(), wrapper, &mockSecretStore{})
			if (err != nil) != tt.wantErr {
				t.Errorf("BrainTask.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBrainTask_Sync(t *testing.T) {
	tests := []struct {
		name        string
		latestDate  string
		issues      []brain.GitHubIssue
		issueBody   string
		expectError bool
		mockSetup   func(mock sqlmock.Sqlmock)
	}{
		{
			name:       "Success - New Issue Ingested",
			latestDate: "2023-01-01",
			issues: []brain.GitHubIssue{
				{Number: 1, Title: "2023-01-02"},
			},
			issueBody:   "## Thought\n- Test Content",
			expectError: false,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// EnsureSchema
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				// GetLatestEntryDate
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow("2023-01-01"))
				// InsertThought
				mock.ExpectExec("INSERT INTO second_brain").WillReturnResult(sqlmock.NewResult(1, 1))
				// RecordSyncHistory (deferred)
				mock.ExpectExec("INSERT INTO second_brain_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "success", 1, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:       "Success - Already Up To Date",
			latestDate: "2023-01-02",
			issues: []brain.GitHubIssue{
				{Number: 1, Title: "2023-01-01"},
			},
			expectError: false,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// EnsureSchema
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				// GetLatestEntryDate
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow("2023-01-02"))
				// RecordSyncHistory (deferred)
				mock.ExpectExec("INSERT INTO second_brain_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "success", 0, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:        "Failure - EnsureSchema Error",
			expectError: true,
			mockSetup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain").WillReturnError(errors.New("db error"))
				// RecordSyncHistory (deferred)
				mock.ExpectExec("INSERT INTO second_brain_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "failure", 0, "failed to ensure second_brain table: db error").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:        "Failure - Fetch Issues Error",
			expectError: true,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// EnsureSchema
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				// GetLatestEntryDate
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow("2023-01-01"))
				// RecordSyncHistory (deferred)
				mock.ExpectExec("INSERT INTO second_brain_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "failure", 0, "fetch error").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:       "Success - InsertThought Failure",
			latestDate: "2023-01-01",
			issues: []brain.GitHubIssue{
				{Number: 1, Title: "2023-01-02"},
			},
			issueBody:   "## Thought\n- Test Content",
			expectError: false,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// EnsureSchema
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				// GetLatestEntryDate
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow("2023-01-01"))
				// InsertThought (fails)
				mock.ExpectExec("INSERT INTO second_brain").WillReturnError(errors.New("insert error"))
				// RecordSyncHistory (deferred) - processedCount is 0 because insert failed
				mock.ExpectExec("INSERT INTO second_brain_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "success", 0, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:        "Failure - GetLatestEntryDate Error",
			expectError: true,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// EnsureSchema
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				// GetLatestEntryDate (fails)
				mock.ExpectQuery("SELECT COALESCE").WillReturnError(errors.New("query error"))
				// RecordSyncHistory (deferred)
				mock.ExpectExec("INSERT INTO second_brain_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "failure", 0, "query error").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:       "Success - Fetch Body Failure",
			latestDate: "2023-01-01",
			issues: []brain.GitHubIssue{
				{Number: 1, Title: "2023-01-02"},
			},
			expectError: false,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// EnsureSchema
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				// GetLatestEntryDate
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow("2023-01-01"))
				// RecordSyncHistory (deferred) - processedCount is 0
				mock.ExpectExec("INSERT INTO second_brain_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "success", 0, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("failed to open mock sql: %v", err)
			}
			defer db.Close()

			if tt.mockSetup != nil {
				tt.mockSetup(mock)
			}

			wrapper := &postgres.PostgresWrapper{DB: db}
			brainStore := NewBrainStore(wrapper)
			task := &BrainTask{}
			mockAPI := &mockBrainAPI{
				fetchRecentJournalsFn: func(repo string) ([]brain.GitHubIssue, error) {
					if tt.name == "Failure - Fetch Issues Error" {
						return nil, errors.New("fetch error")
					}
					return tt.issues, nil
				},
				fetchIssueBodyFn: func(repo string, number int) (string, error) {
					if tt.name == "Success - Fetch Body Failure" {
						return "", errors.New("body fetch error")
					}
					return tt.issueBody, nil
				},
			}

			err = task.Sync(context.Background(), "test/repo", brainStore, mockAPI)
			if (err != nil) != tt.expectError {
				t.Errorf("BrainTask.Sync() error = %v, expectError %v", err, tt.expectError)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %s", err)
			}
		})
	}
}
