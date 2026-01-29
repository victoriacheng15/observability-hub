package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"db"
	"logger"
	"secrets"
	"system-metrics/collectors"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/shirou/gopsutil/v4/host"
)

// App holds the dependencies for the system-metrics service
type App struct {
	DB          *sql.DB
	HostInfoFn  func() (*host.InfoStat, error)
	HostnameFn  func() (string, error)
	NowFn       func() time.Time
	ConnectDBFn func(driverName string, store secrets.SecretStore) (*sql.DB, error)
}

func main() {
	app := &App{
		HostInfoFn:  host.Info,
		HostnameFn:  os.Hostname,
		NowFn:       time.Now,
		ConnectDBFn: db.ConnectPostgres,
	}

	if err := app.Bootstrap(context.Background()); err != nil {
		slog.Error("bootstrap_failed", "error", err)
		os.Exit(1)
	}
}

// Bootstrap separates the main orchestration from the hard os.Exit calls
func (a *App) Bootstrap(ctx context.Context) error {
	// 1. Setup Logging
	logger.Setup(os.Stdout, "system-metrics")
	slog.Info("Starting System Metrics Collector", "version", "1.0.0")

	// Load .env (current or parent)
	_ = godotenv.Load()
	_ = godotenv.Load("../.env")

	// 2. Initialize Secrets Provider
	secretStore, err := secrets.NewBaoProvider()
	if err != nil {
		return fmt.Errorf("secret_provider_init_failed: %w", err)
	}

	// 3. Database Connection
	if err := a.InitDB("postgres", secretStore); err != nil {
		return fmt.Errorf("db_connection_failed: %w", err)
	}
	defer a.DB.Close()

	if err := a.Run(ctx); err != nil {
		return fmt.Errorf("app_run_failed: %w", err)
	}

	return nil
}

func (a *App) InitDB(driverName string, store secrets.SecretStore) error {
	database, err := a.ConnectDBFn(driverName, store)
	if err != nil {
		return err
	}
	a.DB = database
	slog.Info("db_connected", "database", driverName)
	return nil
}

func (a *App) Run(ctx context.Context) error {
	// 1. Initial Detection
	hInfo, err := a.HostInfoFn()
	if err != nil {
		return fmt.Errorf("host_info_failed: %w", err)
	}
	osName := fmt.Sprintf("%s %s", hInfo.Platform, hInfo.PlatformVersion)

	hostName, _ := a.HostnameFn()
	if hostName == "" {
		hostName = "homelab"
	}

	// 2. Ensure Schema
	if err := a.ensureSchema(ctx); err != nil {
		return err
	}

	// 3. Collect and Store Once
	a.collectAndStore(ctx, hostName, osName)
	return nil
}

func (a *App) collectAndStore(ctx context.Context, hostName string, osName string) {
	now := a.NowFn().UTC().Truncate(time.Second)

	// Collect
	cpu, _ := collectors.GetCPUStats()
	mem, _ := collectors.GetMemoryStats()
	disk, _ := collectors.GetDiskStats()
	net, _ := collectors.GetNetworkStats()

	// Store to DB
	metrics := []struct {
		mType   string
		payload interface{}
	}{
		{"cpu", cpu},
		{"memory", mem},
		{"disk", disk},
		{"network", net},
	}

	var insertErrors []string
	for _, m := range metrics {
		if m.payload == nil {
			continue
		}
		payloadJSON, _ := json.Marshal(m.payload)
		_, err := a.DB.ExecContext(ctx,
			"INSERT INTO system_metrics (time, host, os, metric_type, payload) VALUES ($1, $2, $3, $4, $5)",
			now, hostName, osName, m.mType, payloadJSON,
		)
		if err != nil {
			slog.Error("db_insert_failed", "metric_type", m.mType, "error", err)
			insertErrors = append(insertErrors, err.Error())
		}
	}

	// Log success only at the top of the hour and if no errors
	if now.Minute() == 0 {
		if len(insertErrors) == 0 {
			slog.Info("metrics_collected", "status", "success")
		} else {
			slog.Warn("metrics_collected", "status", "partial_failure", "error_count", len(insertErrors))
		}
	}
}

func (a *App) ensureSchema(ctx context.Context) error {
	_, err := a.DB.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS system_metrics (
			time TIMESTAMPTZ(0) NOT NULL,
			host TEXT NOT NULL,
			os TEXT NOT NULL,
			metric_type TEXT NOT NULL,
			payload JSONB NOT NULL
		);
	`)
	if err != nil {
		return fmt.Errorf("schema_init_failed: %w", err)
	}

	// Enable hypertable if TimescaleDB is available
	_, err = a.DB.ExecContext(ctx, "SELECT create_hypertable('system_metrics', 'time', if_not_exists => true);")
	if err != nil {
		// Just info, as we might be running on standard Postgres
		slog.Info("hypertable_check", "status", "skipped_or_failed", "detail", err)
	}
	return nil
}
