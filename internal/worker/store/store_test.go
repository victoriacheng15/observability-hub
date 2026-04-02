package store

import (
	"context"
	"fmt"
	"testing"
	"time"

	"observability-hub/internal/db/postgres"
)

func TestStore_EnsureSchema(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	s := NewStore(mdb.Wrapper())

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "Success",
			setup: func() {
				mdb.ExpectExec("CREATE TYPE metric_kind AS ENUM").WillReturnResult(mdb.NewResult(0, 0))
				mdb.ExpectExec("CREATE TABLE IF NOT EXISTS analytics_metrics").WillReturnResult(mdb.NewResult(0, 0))
				mdb.ExpectExec("CREATE UNIQUE INDEX IF NOT EXISTS idx_analytics_metrics_idempotency").WillReturnResult(mdb.NewResult(0, 0))
				mdb.ExpectExec("CREATE TABLE IF NOT EXISTS reading_analytics").WillReturnResult(mdb.NewResult(0, 0))
				mdb.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain").WillReturnResult(mdb.NewResult(0, 0))
				mdb.ExpectExec("CREATE TABLE IF NOT EXISTS reading_sync_history").WillReturnResult(mdb.NewResult(0, 0))
				mdb.ExpectExec("CREATE TABLE IF NOT EXISTS second_brain_sync_history").WillReturnResult(mdb.NewResult(0, 0))
				mdb.ExpectExec("SELECT create_hypertable").WillReturnResult(mdb.NewResult(0, 0))
			},
			wantErr: false,
		},
		{
			name: "Failure",
			setup: func() {
				mdb.ExpectExec("CREATE TYPE metric_kind AS ENUM").WillReturnError(fmt.Errorf("db error"))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := s.EnsureSchema(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("EnsureSchema() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_RecordAnalyticsMetric(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	s := NewStore(mdb.Wrapper())
	now := time.Now()

	tests := []struct {
		name      string
		featureID string
		kind      MetricKind
		value     float64
		unit      string
		setup     func()
		wantErr   bool
	}{
		{
			name:      "Success",
			featureID: "test-feature",
			kind:      KindEnergy,
			value:     100.5,
			unit:      "joules",
			setup: func() {
				mdb.ExpectExec("INSERT INTO analytics_metrics").
					WithArgs(now, "test-feature", "energy", 100.5, "joules", mdb.AnyArg()).
					WillReturnResult(mdb.NewResult(1, 1))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := s.RecordAnalyticsMetric(context.Background(), now, tt.featureID, tt.kind, tt.value, tt.unit, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("RecordAnalyticsMetric() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_InsertReadingAnalytics(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	s := NewStore(mdb.Wrapper())

	tests := []struct {
		name    string
		mongoID string
		setup   func()
		wantErr bool
	}{
		{
			name:    "Success",
			mongoID: "id123",
			setup: func() {
				mdb.ExpectExec("INSERT INTO reading_analytics").
					WithArgs("id123", mdb.AnyArg(), "source1", "type1", []byte("{}"), []byte("{}")).
					WillReturnResult(mdb.NewResult(1, 1))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := s.InsertReadingAnalytics(context.Background(), tt.mongoID, time.Now(), "source1", "type1", []byte("{}"), []byte("{}"))
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertReadingAnalytics() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_GetLatestEntryDate(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	s := NewStore(mdb.Wrapper())

	tests := []struct {
		name    string
		setup   func()
		want    string
		wantErr bool
	}{
		{
			name: "Success",
			setup: func() {
				rows := mdb.NewRows([]string{"max"}).AddRow("2026-04-01")
				mdb.ExpectQuery("SELECT COALESCE").WillReturnRows(rows)
			},
			want:    "2026-04-01",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			got, err := s.GetLatestEntryDate(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestEntryDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetLatestEntryDate() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_InsertThought(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	s := NewStore(mdb.Wrapper())

	tests := []struct {
		name     string
		checksum string
		setup    func()
		wantErr  bool
	}{
		{
			name:     "Success",
			checksum: "abc123",
			setup: func() {
				mdb.ExpectExec("INSERT INTO second_brain").
					WithArgs("2026-04-01", "content", "category", mdb.AnyArg(), "context", "abc123", 100).
					WillReturnResult(mdb.NewResult(1, 1))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			err := s.InsertThought(context.Background(), "2026-04-01", "content", "category", []string{"tag1"}, "context", tt.checksum, 100)
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertThought() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
