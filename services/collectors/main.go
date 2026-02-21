package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"collectors"
	"db/postgres"
	"env"
	"secrets"
	"telemetry"
)

const (
	serviceName = "collectors"
	interval    = 15 * time.Minute
	step        = "1m"
)

// PromQL queries from promql-queries.md
const (
	queryCPUUtil   = `100 * (1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle", job="kubernetes-service-endpoints"}[5m])))`
	queryRAMUtil   = `100 * (1 - (node_memory_MemAvailable_bytes{job="kubernetes-service-endpoints"} / node_memory_MemTotal_bytes{job="kubernetes-service-endpoints"}))`
	queryDiskRead  = `sum by (instance) (rate(node_disk_read_bytes_total{job="kubernetes-service-endpoints"}[5m]))`
	queryDiskWrite = `sum by (instance) (rate(node_disk_written_bytes_total{job="kubernetes-service-endpoints"}[5m]))`
	queryNetRecv   = `sum by (instance) (rate(node_network_receive_bytes_total{job="kubernetes-service-endpoints"}[5m]))`
	queryNetSent   = `sum by (instance) (rate(node_network_transmit_bytes_total{job="kubernetes-service-endpoints"}[5m]))`
	queryTemp      = `node_hwmon_temp_celsius{job="kubernetes-service-endpoints"}`
)

// ThanosSource defines the interface for fetching metrics from Thanos.
type ThanosSource interface {
	QueryRange(ctx context.Context, query string, start, end time.Time, step string) ([]collectors.Sample, error)
}

// DataStore defines the interface for persisting metrics.
type DataStore interface {
	RecordMetric(ctx context.Context, t time.Time, hostName, osName, metricType string, payload interface{}) error
}

type App struct {
	Thanos ThanosSource
	Store  DataStore
}

func main() {
	env.Load()

	thanosURL := os.Getenv("THANOS_URL")
	if thanosURL == "" {
		telemetry.Error("THANOS_URL_missing", "error", "THANOS_URL environment variable is required")
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 1. Initialize Telemetry
	shutdown, err := telemetry.Init(ctx, serviceName)
	if err != nil {
		fmt.Printf("Warning: OTel init failed: %v\n", err)
	}
	defer shutdown()

	// 2. Initialize Secrets & DB
	secretStore, err := secrets.NewBaoProvider()
	if err != nil {
		telemetry.Error("secret_provider_init_failed", "error", err)
		os.Exit(1)
	}

	wrapper, err := postgres.ConnectPostgres("postgres", secretStore)
	if err != nil {
		telemetry.Error("db_connection_failed", "error", err)
		os.Exit(1)
	}
	defer wrapper.DB.Close()

	store := NewMetricsStore(wrapper)
	if err := store.EnsureSchema(ctx); err != nil {
		telemetry.Error("schema_init_failed", "error", err)
		os.Exit(1)
	}

	app := &App{
		Thanos: collectors.NewThanosClient(thanosURL),
		Store:  store,
	}

	telemetry.Info("service_started", "interval", interval.String())

	// 3. Main Loop
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediate first batch
	app.runBatch(ctx)

	for {
		select {
		case <-ticker.C:
			app.runBatch(ctx)
		case <-ctx.Done():
			telemetry.Info("service_shutting_down")
			return
		}
	}
}

func (a *App) runBatch(ctx context.Context) {
	tracer := telemetry.GetTracer(serviceName)
	ctx, span := tracer.Start(ctx, "job.collect_batch")
	defer span.End()

	end := time.Now().UTC().Truncate(time.Minute)
	start := end.Add(-interval)

	telemetry.Info("batch_started", "start", start.Format(time.RFC3339), "end", end.Format(time.RFC3339))

	// Fetch and Persist Host Metrics
	a.collectAndStoreHostMetrics(ctx, start, end)

	// Fetch Tailscale State
	a.collectTailscale(ctx)

	telemetry.Info("batch_complete")
}

func (a *App) collectAndStoreHostMetrics(ctx context.Context, start, end time.Time) {
	// 1. Fetch CPU Utilization
	cpuSamples, _ := a.Thanos.QueryRange(ctx, queryCPUUtil, start, end, step)

	// 2. Fetch Temperatures
	tempSamples, _ := a.Thanos.QueryRange(ctx, queryTemp, start, end, step)

	// 3. Fetch RAM
	ramSamples, _ := a.Thanos.QueryRange(ctx, queryRAMUtil, start, end, step)

	// 4. Join and Store
	type hostTime struct {
		host string
		ts   int64
	}

	// Merge CPU and Temp
	cpuData := make(map[hostTime]map[string]interface{})

	for _, s := range cpuSamples {
		instance := "unknown"
		if labels, ok := s.Payload["labels"].(map[string]string); ok {
			instance = strings.Split(labels["instance"], ":")[0]
		}
		key := hostTime{instance, s.Timestamp.Unix()}
		val, _ := strconv.ParseFloat(fmt.Sprintf("%v", s.Payload["value"]), 64)
		cpuData[key] = map[string]interface{}{
			"usage":      val,
			"core_temps": make(map[string]float64),
		}
	}

	for _, s := range tempSamples {
		labels, ok := s.Payload["labels"].(map[string]string)
		if !ok {
			continue
		}
		host := strings.Split(labels["instance"], ":")[0]
		key := hostTime{host, s.Timestamp.Unix()}

		if data, ok := cpuData[key]; ok {
			val, _ := strconv.ParseFloat(fmt.Sprintf("%v", s.Payload["value"]), 64)
			sensor := labels["sensor"]
			label := labels["label"]

			if strings.Contains(label, "Package") {
				data["package_temp"] = val
			} else {
				coreMap := data["core_temps"].(map[string]float64)
				coreMap[sensor] = val
			}
		}
	}

	// Flush Merged CPU
	for key, payload := range cpuData {
		a.Store.RecordMetric(ctx, time.Unix(key.ts, 0), key.host, "Linux (Thanos)", "cpu", payload)
	}

	// Flush RAM
	for _, s := range ramSamples {
		host := "unknown"
		if labels, ok := s.Payload["labels"].(map[string]string); ok {
			host = strings.Split(labels["instance"], ":")[0]
		}
		val, _ := strconv.ParseFloat(fmt.Sprintf("%v", s.Payload["value"]), 64)
		payload := map[string]interface{}{"used_percent": val}
		a.Store.RecordMetric(ctx, s.Timestamp, host, "Linux (Thanos)", "memory", payload)
	}
}

func (a *App) collectTailscale(ctx context.Context) {
	// 1. Funnel Status
	funnel, err := collectors.GetFunnelStatus(ctx)
	if err != nil {
		telemetry.Error("funnel_status_failed", "error", err)
	} else {
		if err := a.Store.RecordMetric(ctx, funnel.Fetched, "homelab", "Linux", "tailscale_funnel", funnel); err != nil {
			telemetry.Error("funnel_db_insert_failed", "error", err)
		}
	}

	// 2. Node Status
	status, err := collectors.GetTailscaleStatus(ctx)
	if err != nil {
		telemetry.Error("tailscale_status_failed", "error", err)
	} else {
		if err := a.Store.RecordMetric(ctx, time.Now(), "homelab", "Linux", "tailscale_node", status); err != nil {
			telemetry.Error("node_status_db_insert_failed", "error", err)
		}
	}
}
