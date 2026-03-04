package main

import (
	"context"
	"fmt"
	"observability-hub/internal/db/postgres"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMetricsStore_EnsureSchema(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	store := NewMetricsStore(mdb.Wrapper())

	t.Run("Success", func(t *testing.T) {
		mdb.Mock.ExpectExec("CREATE TABLE IF NOT EXISTS system_metrics").WillReturnResult(sqlmock.NewResult(0, 0))
		mdb.Mock.ExpectExec("SELECT create_hypertable").WillReturnResult(sqlmock.NewResult(0, 0))

		err := store.EnsureSchema(context.Background())
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Failure", func(t *testing.T) {
		mdb.Mock.ExpectExec("CREATE TABLE IF NOT EXISTS system_metrics").WillReturnError(fmt.Errorf("db error"))

		err := store.EnsureSchema(context.Background())
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestMetricsStore_RecordMetric(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	store := NewMetricsStore(mdb.Wrapper())
	now := time.Now()

	t.Run("Success", func(t *testing.T) {
		mdb.Mock.ExpectExec("INSERT INTO system_metrics").
			WithArgs(now, "host1", "linux", "cpu", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.RecordMetric(context.Background(), now, "host1", "linux", "cpu", map[string]interface{}{"val": 1})
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("Nil Payload", func(t *testing.T) {
		err := store.RecordMetric(context.Background(), now, "host1", "linux", "cpu", nil)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}
