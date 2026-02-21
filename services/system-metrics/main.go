package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"db/postgres"
	"env"
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
		meter := telemetry.GetMeter("system.metrics")
		collTotal, _ = telemetry.NewInt64Counter(meter, "system.metrics.collection.total", "Total collection attempts")
		collErrors, _ = telemetry.NewInt64Counter(meter, "system.metrics.collection.errors", "Total collection errors")
		ready = true
	})
}

type App struct {
	Store            *MetricsStore
	HostInfoFn       func() (*host.InfoStat, error)
	HostnameFn       func() (string, error)
	NowFn            func() time.Time
	ConnectDBFn      func(driverName string, store secrets.SecretStore) (*MetricsStore, error)
	SecretProviderFn func() (secrets.SecretStore, error)
}

func main() {
	app := &App{
		HostInfoFn: host.Info,
		HostnameFn: os.Hostname,
		NowFn:      time.Now,
		ConnectDBFn: func(driverName string, store secrets.SecretStore) (*MetricsStore, error) {
			wrapper, err := postgres.ConnectPostgres(driverName, store)
			if err != nil {
				return nil, err
			}
			return NewMetricsStore(wrapper), nil
		},
		SecretProviderFn: func() (secrets.SecretStore, error) {
			return secrets.NewBaoProvider()
		},
	}

	if err := app.Bootstrap(context.Background()); err != nil {
		telemetry.Error("bootstrap_failed", "error", err)
		os.Exit(1)
	}
}

func (a *App) Bootstrap(ctx context.Context) error {
	env.Load()

	// 1. Telemetry Takeover
	shutdown, err := telemetry.Init(ctx, "system.metrics")
	if err != nil {
		telemetry.Warn("otel_init_failed", "error", err)
	}
	defer shutdown()

	ensureMetrics()

	secretStore, err := a.SecretProviderFn()
	if err != nil {
		return fmt.Errorf("secret_provider_init_failed: %w", err)
	}

	if err := a.InitDB("postgres", secretStore); err != nil {
		return fmt.Errorf("db_connection_failed: %w", err)
	}
	telemetry.Info("db_connected")
	defer a.Store.Wrapper.DB.Close()

	return a.Run(ctx)
}

func (a *App) Run(ctx context.Context) error {
	tracer := telemetry.GetTracer("system.metrics")
	ctx, span := tracer.Start(ctx, "job.system_metrics")
	defer span.End()

	if ready {
		telemetry.AddInt64Counter(ctx, collTotal, 1)
	}

	hInfo, err := a.HostInfoFn()
	if err != nil {
		return fmt.Errorf("host_info_failed: %w", err)
	}
	osName := fmt.Sprintf("%s %s", hInfo.Platform, hInfo.PlatformVersion)

	hostName, _ := a.HostnameFn()
	if hostName == "" {
		hostName = "homelab"
	}

	span.SetAttributes(
		telemetry.StringAttribute("os.kernel.version", hInfo.KernelVersion),
	)

	if err := a.Store.EnsureSchema(ctx); err != nil {
		return err
	}

	return a.collectAndStore(ctx, hostName, osName)
}

func (a *App) collectAndStore(ctx context.Context, hostName string, osName string) error {
	tracer := telemetry.GetTracer("system.metrics")
	now := a.NowFn().UTC().Truncate(time.Second)

	// 1. Sample hardware
	_, sSpan := tracer.Start(ctx, "os.poll_stats")
	cpu, _ := metrics.GetCPUStats()
	mem, _ := metrics.GetMemoryStats()
	disk, _ := metrics.GetDiskStats()
	net, _ := metrics.GetNetworkStats()
	sSpan.End()

	// 2. Insert to Postgres
	metricEntries := []struct {
		mType   string
		payload interface{}
	}{
		{"cpu", cpu},
		{"memory", mem},
		{"disk", disk},
		{"network", net},
	}

	// Advanced: Resource Saturation Gauge
	if ready {
		meter := telemetry.GetMeter("system.metrics")
		satGauge, _ := telemetry.NewFloat64ObservableGauge(meter, "system.metrics.resource.saturation", "Resource saturation", func(ctx context.Context, obs telemetry.Float64Observer) error {
			if cpu != nil {
				obs.Observe(cpu.Usage, telemetry.WithMetricAttributes(telemetry.StringAttribute("resource", "cpu")))
			}
			if mem != nil {
				obs.Observe(mem.UsedPercent, telemetry.WithMetricAttributes(telemetry.StringAttribute("resource", "memory")))
			}
			return nil
		})
		_ = satGauge
	}

	var hasError bool
	for _, m := range metricEntries {
		if err := a.Store.RecordMetric(ctx, now, hostName, osName, m.mType, m.payload); err != nil {
			telemetry.Error("db_insert_failed", "metric_type", m.mType, "error", err)
			hasError = true
		}
	}

	if hasError {
		if ready {
			telemetry.AddInt64Counter(ctx, collErrors, 1)
		}
		return fmt.Errorf("partial_collection_failure")
	}

	telemetry.Info("collection_complete", "host", hostName, "duration", time.Since(now).String())
	return nil
}

func (a *App) InitDB(driver string, store secrets.SecretStore) error {
	var err error
	a.Store, err = a.ConnectDBFn(driver, store)
	return err
}
