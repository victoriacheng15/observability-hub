package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"db/postgres"
)

func TestReadingStore_EnsureSchema(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	store := NewReadingStore(mdb.Wrapper())

	t.Run("Success", func(t *testing.T) {
		mdb.ExpectTableCreation("reading_analytics")
		mdb.ExpectTableCreation("reading_sync_history")

		err := store.EnsureSchema(context.Background())
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("First Table Failure", func(t *testing.T) {
		mdb.Mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").
			WillReturnError(errors.New("db error"))

		err := store.EnsureSchema(context.Background())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestReadingStore_RecordSyncHistory(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	store := NewReadingStore(mdb.Wrapper())
	now := time.Now()

	t.Run("Success", func(t *testing.T) {
		mdb.Mock.ExpectExec("INSERT INTO reading_sync_history").
			WithArgs(now, now, "success", 10, "").
			WillReturnResult(mdb.NewResult(1, 1))

		err := store.RecordSyncHistory(context.Background(), now, now, "success", 10, "")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestReadingStore_InsertReadingAnalytics(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	store := NewReadingStore(mdb.Wrapper())

	t.Run("Success", func(t *testing.T) {
		mdb.Mock.ExpectExec("INSERT INTO reading_analytics").
			WithArgs("mongo123", "2026-01-01", "source1", "type1", mdb.AnyArg(), mdb.AnyArg()).
			WillReturnResult(mdb.NewResult(1, 1))

		err := store.InsertReadingAnalytics(context.Background(), "mongo123", "2026-01-01", "source1", "type1", []byte("{}"), []byte("{}"))
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestMongoStoreWrapper_Mock(t *testing.T) {
	docID := "507f1f77bcf86cd799439011"

	t.Run("FetchArticles", func(t *testing.T) {
		// Directly testing the interface implementation logic
		mAPI := &mockMongoStore{
			FetchFn: func(ctx context.Context, limit int64) ([]ReadingDocument, error) {
				return []ReadingDocument{{ID: docID}}, nil
			},
		}
		docs, _ := mAPI.FetchIngestedArticles(context.Background(), 10)
		if len(docs) != 1 || docs[0].ID != docID {
			t.Errorf("mock failed to return expected document")
		}
	})
}
