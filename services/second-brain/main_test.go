package main

import (
	"context"
	"errors"
	"testing"

	"brain"
	"db/postgres"
	"secrets"

	"github.com/DATA-DOG/go-sqlmock"
)

type mockSecretStore struct{}

func (m *mockSecretStore) GetSecret(path, key, fallback string) string { return fallback }
func (m *mockSecretStore) Close() error                                { return nil }

type mockBrainAPI struct {
	fetchRecentFn func(repo string) ([]brain.GitHubIssue, error)
	fetchBodyFn   func(repo string, number int) (string, error)
}

func (m *mockBrainAPI) FetchRecentJournals(repo string) ([]brain.GitHubIssue, error) {
	if m.fetchRecentFn != nil {
		return m.fetchRecentFn(repo)
	}
	return nil, nil
}
func (m *mockBrainAPI) FetchIssueBody(repo string, number int) (string, error) {
	if m.fetchBodyFn != nil {
		return m.fetchBodyFn(repo, number)
	}
	return "", nil
}

func TestApp_Run(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()

	tests := []struct {
		name      string
		repo      string
		secretErr error
		pgErr     error
		wantErr   bool
	}{
		{"Success", "test-repo", nil, nil, false},
		{"No Repo", "", nil, nil, true},
		{"Secret Failure", "test-repo", errors.New("secret error"), nil, true},
		{"Postgres Failure", "test-repo", nil, errors.New("pg error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "Success" {
				mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow("2026-01-01"))
			}

			app := &App{
				SecretProviderFn: func() (secrets.SecretStore, error) {
					return &mockSecretStore{}, tt.secretErr
				},
				PostgresConnFn: func(driver string, store secrets.SecretStore) (*postgres.BrainStore, error) {
					return postgres.NewBrainStore(dbConn), tt.pgErr
				},
				BrainAPI: &mockBrainAPI{},
			}

			t.Setenv("JOURNAL_REPO", tt.repo)

			err := app.Run(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestApp_Sync(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()
	brainStore := postgres.NewBrainStore(dbConn)

	t.Run("Sync Success", func(t *testing.T) {
		repo := "owner/repo"
		latestDate := "2026-02-18"
		newDate := "2026-02-19"

		mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(latestDate))
		mock.ExpectExec("INSERT INTO second_brain").WillReturnResult(sqlmock.NewResult(1, 1))

		app := &App{
			BrainAPI: &mockBrainAPI{
				fetchRecentFn: func(repo string) ([]brain.GitHubIssue, error) {
					return []brain.GitHubIssue{
						{Number: 1, Title: newDate},
					}, nil
				},
				fetchBodyFn: func(repo string, number int) (string, error) {
					return `## Thought
- New thought`, nil
				},
			},
		}

		err := app.Sync(context.Background(), repo, brainStore)
		if err != nil {
			t.Errorf("Sync() failed: %v", err)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unmet mock expectations: %v", err)
		}
	})

	t.Run("Sync Already Up To Date", func(t *testing.T) {
		repo := "owner/repo"
		latestDate := "2026-02-19"

		mock.ExpectQuery("SELECT COALESCE").WillReturnRows(sqlmock.NewRows([]string{"max"}).AddRow(latestDate))

		app := &App{
			BrainAPI: &mockBrainAPI{
				fetchRecentFn: func(repo string) ([]brain.GitHubIssue, error) {
					return []brain.GitHubIssue{
						{Number: 1, Title: "2026-02-18"},
					}, nil
				},
			},
		}

		err := app.Sync(context.Background(), repo, brainStore)
		if err != nil {
			t.Errorf("Sync() failed: %v", err)
		}
	})
}
