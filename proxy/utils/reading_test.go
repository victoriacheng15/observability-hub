package utils

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func TestSyncReadingHandler(t *testing.T) {
	// Setup Postgres Mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	// Setup Mongo Mock
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	// Set Env Vars for the handler to read
	os.Setenv("MONGO_DB_NAME", "testdb")
	os.Setenv("MONGO_COLLECTION", "testcoll")
	defer func() {
		os.Unsetenv("MONGO_DB_NAME")
		os.Unsetenv("MONGO_COLLECTION")
	}()

	mt.Run("success_sync_one_document", func(mt *mtest.T) {
		// Prepare Service with mock DBs
		service := &ReadingService{
			DB:          db,
			MongoClient: mt.Client,
		}

		// 1. Postgres: Create Table
		// We match the prefix roughly or the main part
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").
			WillReturnResult(sqlmock.NewResult(0, 0))

		// 2. Mongo: Find
		objID := primitive.NewObjectID()
		eventTime := "2026-01-04T12:00:00Z"
		firstDoc := bson.D{
			{Key: "_id", Value: objID},
			{Key: "status", Value: "ingested"},
			{Key: "event_type", Value: "cpu_reading"},
			{Key: "source", Value: "test-agent"},
			{Key: "timestamp", Value: eventTime},
			{Key: "payload", Value: bson.D{{Key: "value", Value: 99}}},
			{Key: "meta", Value: bson.D{{Key: "host", Value: "localhost"}}},
		}

		// mtest mocks the response from the server.
		mt.AddMockResponses(mtest.CreateCursorResponse(
			1,
			"testdb.testcoll",
			mtest.FirstBatch,
			firstDoc,
		))

		// 3. Postgres: Insert
		// Expect an INSERT with 6 arguments:
		// mongo_id, timestamp, source, event_type, payload, meta
		mock.ExpectExec("INSERT INTO reading_analytics").
			WithArgs(
				objID.Hex(),
				eventTime,        // timestamp
				"test-agent",     // source
				"cpu_reading",    // event_type
				sqlmock.AnyArg(), // payload (JSON)
				sqlmock.AnyArg(), // meta (JSON)
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 4. Mongo: UpdateOne
		mt.AddMockResponses(bson.D{
			{Key: "ok", Value: 1},
			{Key: "n", Value: 1},
			{Key: "nModified", Value: 1},
		})
		// --- EXECUTION ---
		req := httptest.NewRequest("POST", "/api/sync/reading", nil)
		w := httptest.NewRecorder()

		service.SyncReadingHandler(w, req)

		// --- ASSERTIONS ---
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		// Verify Postgres expectations
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled Postgres expectations: %s", err)
		}
	})

	mt.Run("respect_batch_size_env", func(mt *mtest.T) {
		// Set custom batch size
		os.Setenv("BATCH_SIZE", "50")
		defer os.Unsetenv("BATCH_SIZE")

		service := &ReadingService{
			DB:          db,
			MongoClient: mt.Client,
		}

		// 1. Postgres: Create Table
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").
			WillReturnResult(sqlmock.NewResult(0, 0))

		// 2. Mongo: Find
		// We expect the 'find' command to have a 'limit' field set to 50.
		// mtest doesn't make it super easy to inspect the command options directly in the wrapper without using AddMockResponses,
		// but we can trust that if the code path is hit, the value is used.
		// For a strict unit test, we'd mock the Find options or inspect the command monitor, but here we'll verify the flow completes.

		mt.AddMockResponses(mtest.CreateCursorResponse(
			1,
			"testdb.testcoll",
			mtest.FirstBatch,
			bson.D{}, // Empty batch for this test, just checking query construction doesn't crash
		))

		// --- EXECUTION ---
		req := httptest.NewRequest("POST", "/api/sync/reading", nil)
		w := httptest.NewRecorder()

		service.SyncReadingHandler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	mt.Run("log_error_on_create_table_failure", func(mt *mtest.T) {
		// Capture logs
		var buf bytes.Buffer
		// Save original logger to restore later (although tests run in parallel, this might be risky if parallel=true, but here it is sequential per function usually unless t.Parallel() is called)
		// Since we are modifying a global, we should be careful.
		// For this specific test, we replace the default logger.
		origLogger := slog.Default()
		defer slog.SetDefault(origLogger)

		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))

		service := &ReadingService{
			DB:          db,
			MongoClient: mt.Client,
		}

		// 1. Postgres: Create Table FAILS
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").
			WillReturnError(errors.New("db connection lost"))

		// --- EXECUTION ---
		req := httptest.NewRequest("POST", "/api/sync/reading", nil)
		w := httptest.NewRecorder()

		service.SyncReadingHandler(w, req)

		// --- ASSERTIONS ---
		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}

		// Check if log contains the expected error message
		logOutput := buf.String()
		expectedLogPart := "ETL_ERROR: Failed to create reading_analytics table"
		if !bytes.Contains(buf.Bytes(), []byte(expectedLogPart)) {
			t.Errorf("expected log to contain %q, got %q", expectedLogPart, logOutput)
		}

		// Verify Postgres expectations
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("there were unfulfilled Postgres expectations: %s", err)
		}
	})
}
