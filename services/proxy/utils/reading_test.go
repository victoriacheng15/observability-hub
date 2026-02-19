package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"db/mongodb"
	"db/postgres"

	"github.com/DATA-DOG/go-sqlmock"
)

type mockMongoStore struct {
	fetchFn func(ctx context.Context, limit int64) ([]mongodb.ReadingDocument, error)
	markFn  func(ctx context.Context, id string) error
}

func (m *mockMongoStore) FetchIngestedArticles(ctx context.Context, limit int64) ([]mongodb.ReadingDocument, error) {
	if m.fetchFn != nil {
		return m.fetchFn(ctx, limit)
	}
	return nil, nil
}

func (m *mockMongoStore) MarkArticleAsProcessed(ctx context.Context, id string) error {
	if m.markFn != nil {
		return m.markFn(ctx, id)
	}
	return nil
}

func TestSyncReadingHandler(t *testing.T) {
	// Setup Postgres Mock
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer dbConn.Close()
	readingStore := postgres.NewReadingStore(dbConn)

	t.Run("success_sync_one_document", func(t *testing.T) {
		docID := "507f1f77bcf86cd799439011"
		mongoMock := &mockMongoStore{
			fetchFn: func(ctx context.Context, limit int64) ([]mongodb.ReadingDocument, error) {
				return []mongodb.ReadingDocument{
					{
						ID:        docID,
						Source:    "test-agent",
						Type:      "cpu_reading",
						Timestamp: "2026-01-04T12:00:00Z",
						Payload:   map[string]interface{}{"value": 99},
						Meta:      map[string]interface{}{"host": "localhost"},
					},
				}, nil
			},
			markFn: func(ctx context.Context, id string) error {
				if id != docID {
					t.Errorf("expected id %s, got %s", docID, id)
				}
				return nil
			},
		}

		service := &ReadingService{
			Store:      readingStore,
			MongoStore: mongoMock,
		}

		// Postgres expectations
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO reading_analytics").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO reading_sync_history").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "success", 1, "").
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := httptest.NewRequest("POST", "/api/sync/reading", nil)
		w := httptest.NewRecorder()

		service.SyncReadingHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
		if got := w.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("expected content-type application/json, got %q", got)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response json: %v", err)
		}
		if got := resp["status"]; got != "success" {
			t.Errorf("expected response status success, got %v", got)
		}
		if got := resp["service"]; got != "reading-sync" {
			t.Errorf("expected service reading-sync, got %v", got)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("unfulfilled postgres expectations: %v", err)
		}
	})

	t.Run("mongo_fetch_error", func(t *testing.T) {
		mongoMock := &mockMongoStore{
			fetchFn: func(ctx context.Context, limit int64) ([]mongodb.ReadingDocument, error) {
				return nil, errors.New("mongo error")
			},
		}

		service := &ReadingService{
			Store:      readingStore,
			MongoStore: mongoMock,
		}

		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO reading_sync_history").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "failure", 0, "mongo error").
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := httptest.NewRequest("POST", "/api/sync/reading", nil)
		w := httptest.NewRecorder()

		service.SyncReadingHandler(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if !bytes.Contains(w.Body.Bytes(), []byte("Failed to query Mongo")) {
			t.Errorf("expected body to contain %q, got %q", "Failed to query Mongo", w.Body.String())
		}
	})

	t.Run("respect_batch_size_env", func(t *testing.T) {
		os.Setenv("BATCH_SIZE", "50")
		defer os.Unsetenv("BATCH_SIZE")

		mongoMock := &mockMongoStore{
			fetchFn: func(ctx context.Context, limit int64) ([]mongodb.ReadingDocument, error) {
				if limit != 50 {
					t.Errorf("expected limit 50, got %d", limit)
				}
				return nil, nil
			},
		}

		service := &ReadingService{
			Store:      readingStore,
			MongoStore: mongoMock,
		}

		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_sync_history").WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("INSERT INTO reading_sync_history").WillReturnResult(sqlmock.NewResult(1, 1))

		req := httptest.NewRequest("POST", "/api/sync/reading", nil)
		w := httptest.NewRecorder()

		service.SyncReadingHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode response json: %v", err)
		}
		if got := resp["processed_count"]; got != float64(0) {
			t.Errorf("expected processed_count 0, got %v", got)
		}
	})

	t.Run("log_error_on_create_table_failure", func(t *testing.T) {
		var buf bytes.Buffer
		origLogger := slog.Default()
		defer slog.SetDefault(origLogger)
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

		service := &ReadingService{
			Store:      readingStore,
			MongoStore: &mockMongoStore{},
		}

		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").WillReturnError(errors.New("db error"))
		mock.ExpectExec("INSERT INTO reading_sync_history").
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "failure", 0, "failed to ensure reading_analytics table: db error").
			WillReturnResult(sqlmock.NewResult(1, 1))

		req := httptest.NewRequest("POST", "/api/sync/reading", nil)
		w := httptest.NewRecorder()

		service.SyncReadingHandler(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
		if !bytes.Contains(w.Body.Bytes(), []byte("Failed to ensure database schema")) {
			t.Errorf("expected body to contain %q, got %q", "Failed to ensure database schema", w.Body.String())
		}
	})
}
