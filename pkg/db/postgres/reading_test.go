package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewReadingStore(t *testing.T) {
	mdb, cleanup := NewMockDB(t)
	defer cleanup()

	store := NewReadingStore(mdb.DB)
	if store == nil || store.DB != mdb.DB {
		t.Error("NewReadingStore did not initialize correctly")
	}
}

func TestReadingStore_EnsureSchema(t *testing.T) {
	mdb, cleanup := NewMockDB(t)
	defer cleanup()

	store := NewReadingStore(mdb.DB)

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
	mdb, cleanup := NewMockDB(t)
	defer cleanup()

	store := NewReadingStore(mdb.DB)
	now := time.Now()

	t.Run("Success", func(t *testing.T) {
		mdb.Mock.ExpectExec("INSERT INTO reading_sync_history").
			WithArgs(now, now, "success", 10, "").
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.RecordSyncHistory(context.Background(), now, now, "success", 10, "")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestReadingStore_InsertReadingAnalytics(t *testing.T) {
	mdb, cleanup := NewMockDB(t)
	defer cleanup()

	store := NewReadingStore(mdb.DB)

	t.Run("Success", func(t *testing.T) {
		mdb.Mock.ExpectExec("INSERT INTO reading_analytics").
			WithArgs("mongo123", "2026-01-01", "source1", "type1", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.InsertReadingAnalytics(context.Background(), "mongo123", "2026-01-01", "source1", "type1", []byte("{}"), []byte("{}"))
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
