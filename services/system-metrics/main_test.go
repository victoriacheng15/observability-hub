package main

import (
	"context"
	"errors"
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
	dbMock, _, _ := sqlmock.New()
	defer dbMock.Close()

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
					return postgres.NewMetricsStore(dbMock), tt.err
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
	dbConn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer dbConn.Close()
	store := postgres.NewMetricsStore(dbConn)

	now := time.Date(2026, 1, 29, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		hostInfoErr error
		schemaErr   error
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
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS system_metrics").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("SELECT create_hypertable").
					WillReturnResult(sqlmock.NewResult(0, 0))

				// Expect 4 inserts (cpu, memory, disk, network)
				for i := 0; i < 4; i++ {
					mock.ExpectExec("INSERT INTO system_metrics").
						WithArgs(now, "test-host", "linux 6.0", sqlmock.AnyArg(), sqlmock.AnyArg()).
						WillReturnResult(sqlmock.NewResult(1, 1))
				}
			} else if tt.schemaErr != nil {
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS system_metrics").
					WillReturnError(tt.schemaErr)
			}

			err := app.Run(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unfulfilled expectations: %v", err)
			}
		})
	}
}

func TestApp_Bootstrap(t *testing.T) {
	dbConn, mock, _ := sqlmock.New()
	defer dbConn.Close()

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
					return postgres.NewMetricsStore(dbConn), tt.dbErr
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
				mock.ExpectExec("CREATE TABLE IF NOT EXISTS system_metrics").WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectExec("SELECT create_hypertable").WillReturnResult(sqlmock.NewResult(0, 0))
				for i := 0; i < 4; i++ {
					mock.ExpectExec("INSERT INTO system_metrics").WillReturnResult(sqlmock.NewResult(1, 1))
				}
			}

			err := app.Bootstrap(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("Bootstrap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
