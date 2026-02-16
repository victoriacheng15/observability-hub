package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"db/postgres"
	"logger"
	"metrics"
	"secrets"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/shirou/gopsutil/v4/host"
)

// App holds the dependencies for the system-metrics service
type App struct {
	Store            *postgres.MetricsStore
	HostInfoFn       func() (*host.InfoStat, error)
	HostnameFn       func() (string, error)
	NowFn            func() time.Time
	ConnectDBFn      func(driverName string, store secrets.SecretStore) (*postgres.MetricsStore, error)
	SecretProviderFn func() (secrets.SecretStore, error)
}

func main() {
	app := &App{
		HostInfoFn: host.Info,
		HostnameFn: os.Hostname,
		NowFn:      time.Now,
		ConnectDBFn: func(driverName string, store secrets.SecretStore) (*postgres.MetricsStore, error) {
			conn, err := postgres.ConnectPostgres(driverName, store)
			if err != nil {
				return nil, err
			}
			return postgres.NewMetricsStore(conn), nil
		},
		SecretProviderFn: func() (secrets.SecretStore, error) {
			return secrets.NewBaoProvider()
		},
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
	_ = godotenv.Load("../../.env")

	// 2. Initialize Secrets Provider
	secretStore, err := a.SecretProviderFn()
	if err != nil {
		return fmt.Errorf("secret_provider_init_failed: %w", err)
	}

	// 3. Database Connection
	if err := a.InitDB("postgres", secretStore); err != nil {
		return fmt.Errorf("db_connection_failed: %w", err)
	}
	defer a.Store.DB.Close()

	if err := a.Run(ctx); err != nil {
		return fmt.Errorf("app_run_failed: %w", err)
	}

	return nil
}

func (a *App) InitDB(driverName string, store secrets.SecretStore) error {
	metricsStore, err := a.ConnectDBFn(driverName, store)
	if err != nil {
		return err
	}
	a.Store = metricsStore
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
	if err := a.Store.EnsureSchema(ctx); err != nil {
		return err
	}

	// 3. Collect and Store Once
	a.collectAndStore(ctx, hostName, osName)
	return nil
}

func (a *App) collectAndStore(ctx context.Context, hostName string, osName string) {
	now := a.NowFn().UTC().Truncate(time.Second)

	// Collect
	cpu, _ := metrics.GetCPUStats()
	mem, _ := metrics.GetMemoryStats()
	disk, _ := metrics.GetDiskStats()
	net, _ := metrics.GetNetworkStats()

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
		if err := a.Store.RecordMetric(ctx, now, hostName, osName, m.mType, m.payload); err != nil {
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
