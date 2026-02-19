package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"db/postgres"
	"env"
	"logger"
	"metrics"
	"secrets"
	"telemetry"

	"github.com/shirou/gopsutil/v4/host"
)

var (
	metricsOnce sync.Once
	ready       bool
	collTotal   telemetry.Int64Counter
	collErrors  telemetry.Int64Counter
)

func ensureMetrics() {
	metricsOnce.Do(func() {
		meter := telemetry.GetMeter("system-metrics")
		collTotal, _ = telemetry.NewInt64Counter(meter, "system_metrics.collection.total", "Total collection attempts")
		collErrors, _ = telemetry.NewInt64Counter(meter, "system_metrics.collection.errors", "Total collection errors")
		ready = true
	})
}

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
	// 1. Telemetry Takeover (OTel-native logging, metrics, traces)
	shutdownTracer, shutdownMeter, shutdownLogger, err := telemetry.Init(ctx, "system-metrics")
	if err != nil {
		// Fallback to standard logger if OTel init fails critically
		logger.Setup(os.Stdout, "system-metrics")
		slog.Warn("otel_init_failed, continuing with standard logging", "error", err)
	}
	defer func() {
		sCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownTracer != nil {
			_ = shutdownTracer(sCtx)
		}
		if shutdownMeter != nil {
			_ = shutdownMeter(sCtx)
		}
		if shutdownLogger != nil {
			_ = shutdownLogger(sCtx)
		}
	}()

	slog.Info("Starting System Metrics Collector", "version", "1.1.0")

	env.Load()
	ensureMetrics()

	// 2. Initialize Secrets Provider
	secretStore, err := a.SecretProviderFn()
	if err != nil {
		return fmt.Errorf("secret_provider_init_failed: %w", err)
	}
	defer secretStore.Close()

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
	tracer := telemetry.GetTracer("system-metrics")
	ctx, span := tracer.Start(ctx, "job.collect_metrics")
	defer span.End()

	if ready {
		telemetry.AddInt64Counter(ctx, collTotal, 1)
	}

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
	return a.collectAndStore(ctx, hostName, osName)
}

func (a *App) collectAndStore(ctx context.Context, hostName string, osName string) error {
	tracer := telemetry.GetTracer("system-metrics")
	now := a.NowFn().UTC().Truncate(time.Second)

	// 1. Poll system stats
	_, sSpan := tracer.Start(ctx, "job.poll_system_stats")
	cpu, _ := metrics.GetCPUStats()
	mem, _ := metrics.GetMemoryStats()
	disk, _ := metrics.GetDiskStats()
	net, _ := metrics.GetNetworkStats()
	sSpan.End()

	// 2. Insert to Postgres
	iCtx, iSpan := tracer.Start(ctx, "job.insert_postgres")
	defer iSpan.End()

	metricEntries := []struct {
		mType   string
		payload interface{}
	}{
		{"cpu", cpu},
		{"memory", mem},
		{"disk", disk},
		{"network", net},
	}

	var hasError bool
	for _, m := range metricEntries {
		if err := a.Store.RecordMetric(iCtx, now, hostName, osName, m.mType, m.payload); err != nil {
			slog.Error("db_insert_failed", "metric_type", m.mType, "error", err)
			hasError = true
		}
	}

	if hasError {
		if ready {
			telemetry.AddInt64Counter(ctx, collErrors, 1)
		}
		return fmt.Errorf("partial_collection_failure")
	}

	slog.Info("collection_complete", "host", hostName, "duration", time.Since(now).String())
	return nil
}
