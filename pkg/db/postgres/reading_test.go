package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestNewReadingStore(t *testing.T) {
	dbConn, _, _ := sqlmock.New()
	defer dbConn.Close()

	store := NewReadingStore(dbConn)
	if store == nil || store.DB != dbConn {
		t.Error("NewReadingStore did not initialize correctly")
	}
}

func TestReadingStore_EnsureSchema(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbConn.Close()

	store := NewReadingStore(dbConn)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_sync_history").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := store.EnsureSchema(context.Background())
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("First Table Failure", func(t *testing.T) {
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").
			WillReturnError(errors.New("db error"))

		err := store.EnsureSchema(context.Background())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestReadingStore_RecordSyncHistory(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbConn.Close()

	store := NewReadingStore(dbConn)
	now := time.Now()

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO reading_sync_history").
			WithArgs(now, now, "success", 10, "").
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.RecordSyncHistory(context.Background(), now, now, "success", 10, "")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}

func TestReadingStore_InsertReadingAnalytics(t *testing.T) {
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbConn.Close()

	store := NewReadingStore(dbConn)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO reading_analytics").
			WithArgs("mongo123", "2026-01-01", "source1", "type1", sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.InsertReadingAnalytics(context.Background(), "mongo123", "2026-01-01", "source1", "type1", []byte("{}"), []byte("{}"))
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})
}
