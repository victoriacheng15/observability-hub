package utils

import (
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
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").
			WillReturnResult(sqlmock.NewResult(0, 0))

			// 2. Mongo: Find
		objID := primitive.NewObjectID()
		firstDoc := bson.D{
			{Key: "_id", Value: objID},
			{Key: "status", Value: "ingested"},
			{Key: "event_type", Value: "cpu_reading"},
			{Key: "payload", Value: bson.D{{Key: "value", Value: 99}}},
		}

		// mtest mocks the response from the server.
		// We queue a response for the "find" command which returns a cursor with our document.
		mt.AddMockResponses(mtest.CreateCursorResponse(
			1,
			"testdb.testcoll",
			mtest.FirstBatch,
			firstDoc,
		))

		// 3. Postgres: Insert
		// Expect an INSERT for the document we just "found".
		// We match the arguments. The 3rd arg is JSON blob, so we accept any match for simplicity in this unit test,
		// or we could match specifically.
		mock.ExpectExec("INSERT INTO reading_analytics").
			WithArgs(objID.Hex(), "cpu_reading", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// 4. Mongo: UpdateOne
		// The handler updates the status to "processed".
		// mtest expects an "update" command to be sent. We queue a success response.
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
}
