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

// PromQL queries optimized for Grafana dashboard compatibility
const (
	queryCPUUtil  = `100 * (1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle", job="kubernetes-service-endpoints"}[5m])))`
	queryRAMUtil  = `100 * (1 - (node_memory_MemAvailable_bytes{job="kubernetes-service-endpoints"} / node_memory_MemTotal_bytes{job="kubernetes-service-endpoints"}))`
	queryDiskUsed = `100 * (1 - node_filesystem_avail_bytes{mountpoint="/", job="kubernetes-service-endpoints"} / node_filesystem_size_bytes{mountpoint="/", job="kubernetes-service-endpoints"})`
	queryNetRX    = `node_network_receive_bytes_total{device="enp5s0", job="kubernetes-service-endpoints"}`
	queryNetTX    = `node_network_transmit_bytes_total{device="enp5s0", job="kubernetes-service-endpoints"}`
	queryTemp     = `node_hwmon_temp_celsius{job="kubernetes-service-endpoints"}`
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

	// Get Host Metadata directly from mounted host files
	hostName, osName, err := getHostMetadata()
	if err != nil {
		telemetry.Error("host_metadata_detection_failed", "error", err)
		return
	}

	telemetry.Info("batch_started", "start", start.Format(time.RFC3339), "end", end.Format(time.RFC3339), "host", hostName, "os", osName)

	// Fetch and Persist Host Metrics
	a.collectAndStoreHostMetrics(ctx, start, end, hostName, osName)

	// Fetch Tailscale State (Logs/Metrics only, no DB)
	a.collectTailscale(ctx)

	telemetry.Info("batch_complete")
}

func getHostMetadata() (string, string, error) {
	// 1. Get Hostname from mounted host file
	hostnameBytes, err := os.ReadFile("/etc/host_hostname")
	if err != nil {
		return "", "", fmt.Errorf("failed to read /etc/host_hostname: %w", err)
	}
	hostname := strings.TrimSpace(string(hostnameBytes))

	// 2. Get OS Name from mounted host os-release
	osReleaseBytes, err := os.ReadFile("/etc/host_os-release")
	if err != nil {
		return "", "", fmt.Errorf("failed to read /etc/host_os-release: %w", err)
	}

	var name, version string
	lines := strings.Split(string(osReleaseBytes), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "ID=") {
			name = strings.Trim(strings.TrimPrefix(line, "ID="), "\"")
		}
		if strings.HasPrefix(line, "VERSION_ID=") {
			version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), "\"")
		}
	}
	osName := fmt.Sprintf("%s %s", name, version)

	if hostname == "" || name == "" {
		return "", "", fmt.Errorf("detected host metadata is incomplete")
	}

	return hostname, osName, nil
}

func (a *App) collectAndStoreHostMetrics(ctx context.Context, start, end time.Time, hostName, osName string) {
	// 1. Fetch Samples
	cpuSamples, _ := a.Thanos.QueryRange(ctx, queryCPUUtil, start, end, step)
	tempSamples, _ := a.Thanos.QueryRange(ctx, queryTemp, start, end, step)
	ramSamples, _ := a.Thanos.QueryRange(ctx, queryRAMUtil, start, end, step)
	diskSamples, _ := a.Thanos.QueryRange(ctx, queryDiskUsed, start, end, step)
	netRXSamples, _ := a.Thanos.QueryRange(ctx, queryNetRX, start, end, step)
	netTXSamples, _ := a.Thanos.QueryRange(ctx, queryNetTX, start, end, step)

	// 2. Merge CPU and Temp (Robustly by truncating to minute)
	type timeKey int64
	cpuData := make(map[timeKey]map[string]interface{})

	for _, s := range cpuSamples {
		key := timeKey(s.Timestamp.Truncate(time.Minute).Unix())
		val, _ := strconv.ParseFloat(fmt.Sprintf("%v", s.Payload["value"]), 64)
		cpuData[key] = map[string]interface{}{
			"usage":      val,
			"core_temps": make(map[string]float64),
		}
	}

	for _, s := range tempSamples {
		key := timeKey(s.Timestamp.Truncate(time.Minute).Unix())
		if data, ok := cpuData[key]; ok {
			labels := s.Payload["labels"].(map[string]string)
			val, _ := strconv.ParseFloat(fmt.Sprintf("%v", s.Payload["value"]), 64)
			sensor := labels["sensor"]

			// If temp1 and from coretemp chip, treat as package
			if sensor == "temp1" && strings.Contains(labels["chip"], "coretemp") {
				data["package_temp"] = val
			} else {
				coreMap := data["core_temps"].(map[string]float64)
				// Map "tempX" to "core_X" for legacy compatibility (e.g., temp0 -> core_0)
				coreKey := strings.Replace(sensor, "temp", "core_", 1)
				coreMap[coreKey] = val
			}
		}
	}

	// 3. Flush Merged CPU
	for key, payload := range cpuData {
		a.Store.RecordMetric(ctx, time.Unix(int64(key), 0), hostName, osName, "cpu", payload)
	}

	// 4. Flush RAM
	for _, s := range ramSamples {
		val, _ := strconv.ParseFloat(fmt.Sprintf("%v", s.Payload["value"]), 64)
		a.Store.RecordMetric(ctx, s.Timestamp, hostName, osName, "memory", map[string]interface{}{"used_percent": val})
	}

	// 5. Flush Disk (Stored as {"/": {"used_percent": ...}} for Grafana)
	for _, s := range diskSamples {
		val, _ := strconv.ParseFloat(fmt.Sprintf("%v", s.Payload["value"]), 64)
		payload := map[string]interface{}{
			"/": map[string]interface{}{"used_percent": val},
		}
		a.Store.RecordMetric(ctx, s.Timestamp, hostName, osName, "disk", payload)
	}

	// 6. Merge and Flush Network (Stored as {"enp5s0": {"rx_bytes": ..., "tx_bytes": ...}} for Grafana)
	netData := make(map[timeKey]map[string]interface{})
	for _, s := range netRXSamples {
		key := timeKey(s.Timestamp.Truncate(time.Minute).Unix())
		val, _ := strconv.ParseInt(fmt.Sprintf("%v", s.Payload["value"]), 10, 64)
		netData[key] = map[string]interface{}{
			"enp5s0": map[string]interface{}{"rx_bytes": val},
		}
	}
	for _, s := range netTXSamples {
		key := timeKey(s.Timestamp.Truncate(time.Minute).Unix())
		val, _ := strconv.ParseInt(fmt.Sprintf("%v", s.Payload["value"]), 10, 64)
		if n, ok := netData[key]; ok {
			n["enp5s0"].(map[string]interface{})["tx_bytes"] = val
		} else {
			netData[key] = map[string]interface{}{
				"enp5s0": map[string]interface{}{"tx_bytes": val},
			}
		}
	}
	for key, payload := range netData {
		a.Store.RecordMetric(ctx, time.Unix(int64(key), 0), hostName, osName, "network", payload)
	}
}

func (a *App) collectTailscale(ctx context.Context) {
	// 1. Funnel Status
	funnel, err := collectors.GetFunnelStatus(ctx)
	if err != nil {
		telemetry.Error("funnel_status_failed", "error", err)
	} else {
		telemetry.Info("tailscale_funnel_status", "active", funnel.Active, "target", funnel.Target)
	}

	// 2. Node Status
	status, err := collectors.GetTailscaleStatus(ctx)
	if err != nil {
		telemetry.Error("tailscale_status_failed", "error", err)
	} else {
		telemetry.Info("tailscale_node_status", "backend_state", status["BackendState"])
	}
}
