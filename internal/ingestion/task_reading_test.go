package ingestion

import (
	"context"
	"errors"
	"testing"

	"observability-hub/internal/db/postgres"
	"observability-hub/internal/secrets"

	"github.com/DATA-DOG/go-sqlmock"
)

type mockMongoAPI struct {
	fetchIngestedArticlesFn  func(ctx context.Context, limit int64) ([]ReadingDocument, error)
	markArticleAsProcessedFn func(ctx context.Context, id string) error
	closeFn                  func(ctx context.Context) error
}

func (m *mockMongoAPI) FetchIngestedArticles(ctx context.Context, limit int64) ([]ReadingDocument, error) {
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

func TestReadingTask_Name(t *testing.T) {
	task := &ReadingTask{}
	if task.Name() != "reading" {
		t.Errorf("Expected reading, got %s", task.Name())
	}
}

func TestReadingTask_Run(t *testing.T) {
	tests := []struct {
		name     string
		mongoErr error
		wantErr  bool
	}{
		{
			name:     "Mongo Connection Failure",
			mongoErr: errors.New("connection failed"),
			wantErr:  true,
		},
		{
			name:     "Success",
			mongoErr: nil,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, _ := sqlmock.New()
			defer db.Close()
			wrapper := &postgres.PostgresWrapper{DB: db}

			oldNewMongoStore := newMongoStore
			defer func() { newMongoStore = oldNewMongoStore }()
			newMongoStore = func(store secrets.SecretStore) (MongoStoreAPI, error) {
				if tt.mongoErr != nil {
					return nil, tt.mongoErr
				}
				return &mockMongoAPI{
					fetchIngestedArticlesFn: func(ctx context.Context, limit int64) ([]ReadingDocument, error) {
						return nil, nil
					},
				}, nil
			}

			if !tt.wantErr {
				// Expectations for Sync
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("INSERT INTO reading_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "success", 0, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
			}

			task := &ReadingTask{}
			err := task.Run(context.Background(), wrapper, &mockSecretStore{})
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadingTask.Run() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestReadingTask_Sync(t *testing.T) {
	tests := []struct {
		name        string
		articles    []ReadingDocument
		expectError bool
		mockSetup   func(mock sqlmock.Sqlmock)
	}{
		{
			name: "Success - New Articles Synced",
			articles: []ReadingDocument{
				{ID: "1", Source: "test", Type: "article", Timestamp: "2023-01-01", Payload: map[string]any{}, Meta: map[string]any{}},
			},
			expectError: false,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// EnsureSchema
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				// InsertReadingAnalytics
				mock.ExpectExec("INSERT INTO reading_analytics").WillReturnResult(sqlmock.NewResult(1, 1))
				// RecordSyncHistory (deferred)
				mock.ExpectExec("INSERT INTO reading_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "success", 1, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:        "Success - No Articles",
			articles:    []ReadingDocument{},
			expectError: false,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// EnsureSchema
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				// RecordSyncHistory (deferred)
				mock.ExpectExec("INSERT INTO reading_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "success", 0, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name:        "Failure - Fetch Error",
			articles:    nil,
			expectError: true,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// EnsureSchema
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				// RecordSyncHistory (deferred)
				mock.ExpectExec("INSERT INTO reading_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "failure", 0, "mongo error").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "Success - With BATCH_SIZE",
			articles: []ReadingDocument{
				{ID: "1", Source: "test", Type: "article", Timestamp: "2023-01-01", Payload: map[string]any{}, Meta: map[string]any{}},
			},
			expectError: false,
			mockSetup: func(mock sqlmock.Sqlmock) {
				t.Setenv("BATCH_SIZE", "50")
				// EnsureSchema
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				// InsertReadingAnalytics
				mock.ExpectExec("INSERT INTO reading_analytics").WillReturnResult(sqlmock.NewResult(1, 1))
				// RecordSyncHistory (deferred)
				mock.ExpectExec("INSERT INTO reading_sync_history").
					WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "success", 1, "").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "Success - MarkProcessed Failure",
			articles: []ReadingDocument{
				{ID: "1", Source: "test", Type: "article", Timestamp: "2023-01-01", Payload: map[string]any{}, Meta: map[string]any{}},
			},
			expectError: false,
			mockSetup: func(mock sqlmock.Sqlmock) {
				// EnsureSchema
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
				// InsertReadingAnalytics
				mock.ExpectExec("INSERT INTO reading_analytics").WillReturnResult(sqlmock.NewResult(1, 1))
				// RecordSyncHistory (deferred) - processedCount will be 0 because MarkProcessed failed
				mock.ExpectExec("INSERT INTO reading_sync_history").
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
			readingStore := NewReadingStore(wrapper)
			task := &ReadingTask{}
			mockMStore := &mockMongoAPI{
				fetchIngestedArticlesFn: func(ctx context.Context, limit int64) ([]ReadingDocument, error) {
					if tt.name == "Success - With BATCH_SIZE" && limit != 50 {
						t.Errorf("Expected limit 50, got %d", limit)
					}
					if tt.articles == nil && tt.expectError {
						return nil, errors.New("mongo error")
					}
					return tt.articles, nil
				},
				markArticleAsProcessedFn: func(ctx context.Context, id string) error {
					if tt.name == "Success - MarkProcessed Failure" {
						return errors.New("mongo update error")
					}
					return nil
				},
			}

			err = task.Sync(context.Background(), readingStore, mockMStore)
			if (err != nil) != tt.expectError {
				t.Errorf("ReadingTask.Sync() error = %v, expectError %v", err, tt.expectError)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %s", err)
			}
		})
	}
}
