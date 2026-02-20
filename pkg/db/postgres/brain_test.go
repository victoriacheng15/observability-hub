package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestBrainStore_GetLatestEntryDate(t *testing.T) {
	mdb, cleanup := NewMockDB(t)
	defer cleanup()

	store := NewBrainStore(mdb.DB)

	t.Run("Success with date", func(t *testing.T) {
		mdb.Mock.ExpectQuery("SELECT COALESCE").
			WillReturnRows(sqlmock.NewRows([]string{"date"}).AddRow("2026-02-16"))

		date, err := store.GetLatestEntryDate(context.Background())
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if date != "2026-02-16" {
			t.Errorf("expected 2026-02-16, got %s", date)
		}
	})

	t.Run("Success with default", func(t *testing.T) {
		mdb.Mock.ExpectQuery("SELECT COALESCE").
			WillReturnRows(sqlmock.NewRows([]string{"date"}).AddRow("1970-01-01"))

		date, err := store.GetLatestEntryDate(context.Background())
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if date != "1970-01-01" {
			t.Errorf("expected 1970-01-01, got %s", date)
		}
	})

	t.Run("Database error", func(t *testing.T) {
		mdb.Mock.ExpectQuery("SELECT COALESCE").
			WillReturnError(errors.New("db error"))

		_, err := store.GetLatestEntryDate(context.Background())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestBrainStore_GetPARAStats(t *testing.T) {
	mdb, cleanup := NewMockDB(t)
	defer cleanup()

	store := NewBrainStore(mdb.DB)

	t.Run("Success", func(t *testing.T) {
		mdb.Mock.ExpectQuery("SELECT category, total_entries, latest_entry FROM second_brain_stats").
			WillReturnRows(sqlmock.NewRows([]string{"category", "total_entries", "latest_entry"}).
				AddRow("project", 5, "2026-02-16").
				AddRow("area", 10, "2026-02-15"))

		stats, err := store.GetPARAStats(context.Background())
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if len(stats) != 2 {
			t.Errorf("expected 2 stats, got %d", len(stats))
		}
		if stats[0].Category != "project" || stats[0].TotalCount != 5 {
			t.Errorf("stat 0 mismatch: %+v", stats[0])
		}
	})

	t.Run("Database error", func(t *testing.T) {
		mdb.Mock.ExpectQuery("SELECT category, total_entries, latest_entry FROM second_brain_stats").
			WillReturnError(errors.New("db error"))

		_, err := store.GetPARAStats(context.Background())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestBrainStore_InsertThought(t *testing.T) {
	mdb, cleanup := NewMockDB(t)
	defer cleanup()

	store := NewBrainStore(mdb.DB)

	t.Run("Success", func(t *testing.T) {
		mdb.Mock.ExpectExec("INSERT INTO second_brain").
			WithArgs("2026-02-16", "content", "resource", sqlmock.AnyArg(), "context", "chksum", 100).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.InsertThought(context.Background(), "2026-02-16", "content", "resource", []string{"tag1"}, "context", "chksum", 100)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Database error", func(t *testing.T) {
		mdb.Mock.ExpectExec("INSERT INTO second_brain").
			WillReturnError(errors.New("insert failed"))

		err := store.InsertThought(context.Background(), "2026-02-16", "content", "resource", []string{"tag1"}, "context", "chksum", 100)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
