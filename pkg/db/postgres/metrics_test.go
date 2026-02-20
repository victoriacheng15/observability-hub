package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestMetricsStore_EnsureSchema(t *testing.T) {
	mdb, cleanup := NewMockDB(t)
	defer cleanup()

	store := NewMetricsStore(mdb.DB)

	t.Run("Success", func(t *testing.T) {
		mdb.ExpectTableCreation("system_metrics")
		mdb.ExpectHypertableCreation("system_metrics")

		err := store.EnsureSchema(context.Background())
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Table Creation Failure", func(t *testing.T) {
		mdb.Mock.ExpectExec("CREATE TABLE IF NOT EXISTS system_metrics").
			WillReturnError(errors.New("db error"))

		err := store.EnsureSchema(context.Background())
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("Hypertable Failure (Ignored)", func(t *testing.T) {
		mdb.ExpectTableCreation("system_metrics")
		mdb.Mock.ExpectExec("SELECT create_hypertable").
			WillReturnError(errors.New("timescale not available"))

		err := store.EnsureSchema(context.Background())
		if err != nil {
			t.Errorf("expected no error (hypertable failure should be logged but ignored), got %v", err)
		}
	})
}

func TestMetricsStore_RecordMetric(t *testing.T) {
	mdb, cleanup := NewMockDB(t)
	defer cleanup()

	store := NewMetricsStore(mdb.DB)
	now := time.Now()

	t.Run("Success", func(t *testing.T) {
		payload := map[string]interface{}{"cpu": 10.5}
		mdb.Mock.ExpectExec("INSERT INTO system_metrics").
			WithArgs(now, "host1", "linux", "cpu", sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := store.RecordMetric(context.Background(), now, "host1", "linux", "cpu", payload)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("Nil Payload", func(t *testing.T) {
		err := store.RecordMetric(context.Background(), now, "host1", "linux", "cpu", nil)
		if err != nil {
			t.Errorf("expected no error for nil payload, got %v", err)
		}
	})

	t.Run("Insert Failure", func(t *testing.T) {
		payload := map[string]interface{}{"cpu": 10.5}
		mdb.Mock.ExpectExec("INSERT INTO system_metrics").
			WillReturnError(errors.New("insert failed"))

		err := store.RecordMetric(context.Background(), now, "host1", "linux", "cpu", payload)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})
}
