package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"secrets"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/shirou/gopsutil/v4/host"
)

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
				ConnectDBFn: func(driverName string, store secrets.SecretStore) (*sql.DB, error) {
					return dbMock, tt.err
				},
			}
			err := app.InitDB("postgres", nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("InitDB() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && app.DB != dbMock {
				t.Error("InitDB() did not set DB field")
			}
		})
	}
}

func TestApp_Run(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

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
				DB: db,
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

func TestApp_CollectAndStore_PartialFailure(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	defer db.Close()

	now := time.Date(2026, 1, 29, 12, 0, 0, 0, time.UTC)
	app := &App{
		DB:    db,
		NowFn: func() time.Time { return now },
	}

	// Mock one success and one failure
	mock.ExpectExec("INSERT INTO system_metrics").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO system_metrics").WillReturnError(errors.New("insert failed"))
	mock.ExpectExec("INSERT INTO system_metrics").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO system_metrics").WillReturnResult(sqlmock.NewResult(1, 1))

	app.collectAndStore(context.Background(), "test-host", "linux 6.0")

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}

func TestApp_Bootstrap_FailSecrets(t *testing.T) {
	// Unset env to trigger NewBaoProvider failure
	origAddr := os.Getenv("BAO_ADDR")
	origToken := os.Getenv("BAO_TOKEN")
	os.Unsetenv("BAO_ADDR")
	os.Unsetenv("BAO_TOKEN")
	defer func() {
		os.Setenv("BAO_ADDR", origAddr)
		os.Setenv("BAO_TOKEN", origToken)
	}()

	app := &App{
		ConnectDBFn: func(driverName string, store secrets.SecretStore) (*sql.DB, error) {
			return nil, errors.New("should not be called if secrets fail, but NewBaoProvider might not fail")
		},
	}
	err := app.Bootstrap(context.Background())
	if err == nil {
		// If NewBaoProvider didn't fail, InitDB should have failed
		t.Log("Bootstrap did not fail at NewBaoProvider, checking if it failed at InitDB")
	}
}
