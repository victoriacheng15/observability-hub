package main

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"db/postgres"
	"secrets"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/shirou/gopsutil/v4/host"
)

// mockSecretStore implements secrets.SecretStore for testing
type mockSecretStore struct{}

func (m *mockSecretStore) GetSecret(path, key, fallback string) string { return fallback }
func (m *mockSecretStore) Close() error                                { return nil }

func TestApp_InitDB(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	tests := []struct {
		name    string
		err     error
		wantErr bool
	}{
		{
			name:    "Success",
			err:     nil,
			wantErr: false,
		},
		{
			name:    "Connection Failure",
			err:     errors.New("connection failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				ConnectDBFn: func(driverName string, store secrets.SecretStore) (*postgres.MetricsStore, error) {
					return postgres.NewMetricsStore(mdb.DB), tt.err
				},
			}
			err := app.InitDB("postgres", &mockSecretStore{})
			if (err != nil) != tt.wantErr {
				t.Errorf("InitDB() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && app.Store == nil {
				t.Error("InitDB() did not set Store field")
			}
		})
	}
}

func TestApp_Run(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()
	store := postgres.NewMetricsStore(mdb.DB)

	// Use a fixed time for stability
	now := time.Date(2026, 1, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		hostInfoErr error
		schemaErr   error
		dbErr       error
		wantErr     bool
	}{
		{
			name:    "Success",
			wantErr: false,
		},
		{
			name:        "Host Info Failure",
			hostInfoErr: errors.New("host info error"),
			wantErr:     true,
		},
		{
			name:      "Schema Init Failure",
			schemaErr: errors.New("schema error"),
			wantErr:   true,
		},
		{
			name:    "Partial Collection Failure",
			dbErr:   errors.New("insert failed"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				Store: store,
				HostInfoFn: func() (*host.InfoStat, error) {
					if tt.hostInfoErr != nil {
						return nil, tt.hostInfoErr
					}
					return &host.InfoStat{Platform: "linux", PlatformVersion: "6.0"}, nil
				},
				HostnameFn: func() (string, error) {
					return "test-host", nil
				},
				NowFn: func() time.Time {
					return now
				},
			}

			if !tt.wantErr {
				// Expectations for Success
				mdb.ExpectTableCreation("system_metrics")
				mdb.ExpectHypertableCreation("system_metrics")

				// Expect 4 inserts (cpu, memory, disk, network)
				for i := 0; i < 4; i++ {
					mdb.Mock.ExpectExec("INSERT INTO system_metrics").
						WithArgs(now, "test-host", "linux 6.0", sqlmock.AnyArg(), sqlmock.AnyArg()).
						WillReturnResult(sqlmock.NewResult(1, 1))
				}
			} else if tt.schemaErr != nil {
				mdb.Mock.ExpectExec("CREATE TABLE IF NOT EXISTS system_metrics").
					WillReturnError(tt.schemaErr)
			} else if tt.dbErr != nil {
				mdb.ExpectTableCreation("system_metrics")
				mdb.ExpectHypertableCreation("system_metrics")
				// Fail the first insert
				mdb.Mock.ExpectExec("INSERT INTO system_metrics").WillReturnError(tt.dbErr)
				// Remaining inserts still proceed but error is bubbled
				for i := 0; i < 3; i++ {
					mdb.Mock.ExpectExec("INSERT INTO system_metrics").WillReturnResult(sqlmock.NewResult(1, 1))
				}
			}

			err := app.Run(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mdb.Mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestApp_Bootstrap(t *testing.T) {
	mdb, cleanup := postgres.NewMockDB(t)
	defer cleanup()

	// Ensure OTel doesn't try to connect to a real endpoint during tests
	origEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:30317")
	defer os.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", origEndpoint)

	tests := []struct {
		name      string
		secretErr error
		dbErr     error
		wantErr   bool
	}{
		{"Success", nil, nil, false},
		{"Secret Failure", errors.New("secret error"), nil, true},
		{"DB Failure", nil, errors.New("db error"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &App{
				SecretProviderFn: func() (secrets.SecretStore, error) {
					return &mockSecretStore{}, tt.secretErr
				},
				ConnectDBFn: func(driverName string, store secrets.SecretStore) (*postgres.MetricsStore, error) {
					return postgres.NewMetricsStore(mdb.DB), tt.dbErr
				},
				HostInfoFn: func() (*host.InfoStat, error) {
					return &host.InfoStat{Platform: "linux", PlatformVersion: "6.0"}, nil
				},
				HostnameFn: func() (string, error) {
					return "test-host", nil
				},
				NowFn: func() time.Time {
					return time.Now()
				},
			}

			if !tt.wantErr {
				mdb.ExpectTableCreation("system_metrics")
				mdb.ExpectHypertableCreation("system_metrics")
				for i := 0; i < 4; i++ {
					mdb.Mock.ExpectExec("INSERT INTO system_metrics").WillReturnResult(sqlmock.NewResult(1, 1))
				}
			}

			err := app.Bootstrap(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
