package collectors

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

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			setup: func() {
				mdb.Mock.ExpectExec("CREATE TABLE IF NOT EXISTS system_metrics").WillReturnResult(sqlmock.NewResult(0, 0))
				mdb.Mock.ExpectExec("SELECT create_hypertable").WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name: "Failure",
			setup: func() {
				mdb.Mock.ExpectExec("CREATE TABLE IF NOT EXISTS system_metrics").WillReturnError(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := store.EnsureSchema(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetricsStore_RecordMetric(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	store := NewMetricsStore(mdb.Wrapper())
	now := time.Now()

	tests := []struct {
		name    string
		setup   func()
		payload interface{}
		wantErr bool
	}{
		{
			name: "Success",
			setup: func() {
				mdb.Mock.ExpectExec("INSERT INTO system_metrics").
					WithArgs(now, "host1", "linux", "cpu", sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
			payload: map[string]interface{}{"val": 1},
			wantErr: false,
		},
		{
			name:    "Nil Payload",
			setup:   func() {},
			payload: nil,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := store.RecordMetric(context.Background(), now, "host1", "linux", "cpu", tt.payload)
			if (err != nil) != tt.wantErr {
				t.Errorf("RecordMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
