package ingestion

import (
	"context"
	"testing"
	"time"

	"observability-hub/internal/db/postgres"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestRecordSyncHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock sql: %v", err)
	}
	defer db.Close()

	wrapper := &postgres.PostgresWrapper{DB: db}
	ctx := context.Background()
	startTime := time.Now()
	endTime := startTime.Add(time.Minute)

	mock.ExpectExec("INSERT INTO test_table").
		WithArgs(startTime, endTime, "success", 10, "").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = RecordSyncHistory(ctx, wrapper, "test_table", startTime, endTime, "success", 10, "")
	if err != nil {
		t.Errorf("RecordSyncHistory() error = %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %s", err)
	}
}

func TestEnsureHistorySchema(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock sql: %v", err)
	}
	defer db.Close()

	wrapper := &postgres.PostgresWrapper{DB: db}
	ctx := context.Background()

	mock.ExpectExec("CREATE TABLE IF NOT EXISTS test_table").
		WillReturnResult(sqlmock.NewResult(0, 0))

	err = EnsureHistorySchema(ctx, wrapper, "test_table")
	if err != nil {
		t.Errorf("EnsureHistorySchema() error = %v", err)
	}
}

func TestReadingStore(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock sql: %v", err)
	}
	defer db.Close()

	wrapper := &postgres.PostgresWrapper{DB: db}
	store := NewReadingStore(wrapper)
	ctx := context.Background()

	t.Run("EnsureSchema", func(t *testing.T) {
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS reading_sync_history").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := store.EnsureSchema(ctx)
		if err != nil {
			t.Errorf("EnsureSchema() error = %v", err)
		}
	})

	t.Run("InsertReadingAnalytics", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO reading_analytics").
			WithArgs("mongo-id", "2023-01-01", "source", "type", []byte("{}"), []byte("{}")).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.InsertReadingAnalytics(ctx, "mongo-id", "2023-01-01", "source", "type", []byte("{}"), []byte("{}"))
		if err != nil {
			t.Errorf("InsertReadingAnalytics() error = %v", err)
		}
	})
}

func TestBrainStore(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to open mock sql: %v", err)
	}
	defer db.Close()

	wrapper := &postgres.PostgresWrapper{DB: db}
	store := NewBrainStore(wrapper)
	ctx := context.Background()

	t.Run("EnsureSchema", func(t *testing.T) {
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain_sync_history").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := store.EnsureSchema(ctx)
		if err != nil {
			t.Errorf("EnsureSchema() error = %v", err)
		}
	})

	t.Run("GetLatestEntryDate", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"max"}).AddRow("2023-01-01")
		mock.ExpectQuery("SELECT COALESCE").WillReturnRows(rows)

		date, err := store.GetLatestEntryDate(ctx)
		if err != nil {
			t.Errorf("GetLatestEntryDate() error = %v", err)
		}
		if date != "2023-01-01" {
			t.Errorf("GetLatestEntryDate() got = %v, want 2023-01-01", date)
		}
	})

	t.Run("InsertThought", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO second_brain").
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.InsertThought(ctx, "2023-01-01", "content", "category", []string{"tag1"}, "context", "checksum", 100)
		if err != nil {
			t.Errorf("InsertThought() error = %v", err)
		}
	})
}
