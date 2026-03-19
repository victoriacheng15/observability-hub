package analytics

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"observability-hub/internal/telemetry"
)

const (
	ServiceName = "analytics"
	Interval    = 15 * time.Minute
	Step        = "1m"
)

var (
	metricsOnce  sync.Once
	collTotal    telemetry.Int64Counter
	collErrors   telemetry.Int64Counter
	collDuration telemetry.Int64Histogram

	tailscaleMu     sync.RWMutex
	tailscaleActive int64

	pathHostname  = "/etc/host_hostname"
	pathOSRelease = "/etc/host_os-release"
)

// PromQL queries optimized for Grafana dashboard compatibility
const (
	QueryCPUUtil  = `100 * (1 - avg by (instance) (rate(node_cpu_seconds_total{mode="idle", job="kubernetes-service-endpoints"}[5m])))`
	QueryRAMUtil  = `100 * (1 - (node_memory_MemAvailable_bytes{job="kubernetes-service-endpoints"} / node_memory_MemTotal_bytes{job="kubernetes-service-endpoints"}))`
	QueryDiskUsed = `100 * (1 - node_filesystem_avail_bytes{mountpoint="/", job="kubernetes-service-endpoints"} / node_filesystem_size_bytes{mountpoint="/", job="kubernetes-service-endpoints"})`
	QueryNetRX    = `node_network_receive_bytes_total{device="eno1", job="kubernetes-service-endpoints"}`
	QueryNetTX    = `node_network_transmit_bytes_total{device="eno1", job="kubernetes-service-endpoints"}`
	QueryTemp     = `node_hwmon_temp_celsius{job="kubernetes-service-endpoints"}`

	// Kepler Energy Queries (AMD Hardware Optimized)
	QueryNodeEnergy = `sum(increase(kepler_node_cpu_joules_total[15m]))`
)

// ThanosSource defines the interface for fetching metrics from Thanos.
type ThanosSource interface {
	QueryRange(ctx context.Context, query string, start, end time.Time, step string) ([]Sample, error)
}

// DataStore defines the interface for persisting metrics.
type DataStore interface {
	RecordMetric(ctx context.Context, t time.Time, hostName, osName, metricType string, payload interface{}) error
	RecordAnalyticsMetric(ctx context.Context, t time.Time, featureID string, kind MetricKind, value float64, unit string, metadata map[string]interface{}) error
}

// ResourceProvider defines the interface for fetching raw consumption data.
type ResourceProvider interface {
	GetEnergyJoules(ctx context.Context, start, end time.Time) (float64, error)
	GetContainerEnergy(ctx context.Context, start, end time.Time) (map[string]float64, error)
	GetHostServiceCPU(ctx context.Context, start, end time.Time) (map[string]float64, error)
	GetValueUnits(ctx context.Context, start, end time.Time) (map[string]float64, error)
	GetCarbonIntensity(ctx context.Context) (float64, error) // gCO2 per kWh
	GetCostFactor(ctx context.Context) (float64, error)      // CAD per Joule
}

type Service struct {
	Thanos    ThanosSource
	Store     DataStore
	Resources ResourceProvider
}

func NewService(thanos ThanosSource, store DataStore, resources ResourceProvider) *Service {
	EnsureMetrics()
	return &Service{
		Thanos:    thanos,
		Store:     store,
		Resources: resources,
	}
}

func (s *Service) Start(ctx context.Context) error {
	telemetry.Info("service_started", "interval", Interval.String())

	// Run immediate first batch
	s.RunBatch(ctx)

	// Calculate time until next interval boundary
	now := time.Now().UTC()
	next := now.Truncate(Interval).Add(Interval)
	sleepDuration := next.Sub(now)

	telemetry.Info("waiting_for_time_boundary", "sleep", sleepDuration.String(), "next_run", next.Format(time.RFC3339))

	// Wait for the boundary or shutdown
	select {
	case <-time.After(sleepDuration):
	case <-ctx.Done():
		telemetry.Info("service_shutting_down")
		return nil
	}

	// Now we are aligned, start the regular ticker
	ticker := time.NewTicker(Interval)
	defer ticker.Stop()

	// Run the first aligned batch immediately
	s.RunBatch(ctx)

	for {
		select {
		case <-ticker.C:
			s.RunBatch(ctx)
		case <-ctx.Done():
			telemetry.Info("service_shutting_down")
			return nil
		}
	}
}

func EnsureMetrics() {
	metricsOnce.Do(func() {
		meter := telemetry.GetMeter(ServiceName)
		collTotal, _ = telemetry.NewInt64Counter(meter, "analytics.batch.total", "Total processing batches")
		collErrors, _ = telemetry.NewInt64Counter(meter, "analytics.batch.errors.total", "Total batch errors")
		collDuration, _ = telemetry.NewInt64Histogram(meter, "analytics.batch.duration.ms", "Batch processing duration in milliseconds", "ms")

		_, _ = telemetry.NewInt64ObservableGauge(meter, "analytics.tailscale.active", "Tailscale Funnel active status", func(ctx context.Context, obs telemetry.Int64Observer) error {
			tailscaleMu.RLock()
			obs.Observe(tailscaleActive)
			tailscaleMu.RUnlock()
			return nil
		})
	})
}

func (s *Service) RunBatch(ctx context.Context) {
	start := time.Now()
	tracer := telemetry.GetTracer(ServiceName)
	ctx, span := tracer.Start(ctx, "job.collect_batch")
	defer span.End()

	telemetry.AddInt64Counter(ctx, collTotal, 1)
	defer func() {
		if collDuration != nil {
			telemetry.RecordInt64Histogram(ctx, collDuration, time.Since(start).Milliseconds())
		}
	}()

	end := time.Now().UTC().Truncate(Interval)
	startBoundary := end.Add(-Interval)

	// 1. Get Host Metadata directly from mounted host files
	hostName, osName, err := getHostMetadata()
	if err != nil {
		telemetry.Error("host_metadata_detection_failed", "error", err)
		telemetry.AddInt64Counter(ctx, collErrors, 1)
		return
	}

	telemetry.Info("batch_started", "start", startBoundary.Format(time.RFC3339), "end", end.Format(time.RFC3339), "host", hostName, "os", osName)

	span.SetAttributes(
		telemetry.StringAttribute("host", hostName),
		telemetry.StringAttribute("os", osName),
		telemetry.StringAttribute("start", startBoundary.Format(time.RFC3339)),
		telemetry.StringAttribute("end", end.Format(time.RFC3339)),
	)

	// 2. Fetch and Persist Legacy Host Metrics
	s.collectAndStoreHostMetrics(ctx, startBoundary, end, hostName, osName)

	// 3. Resource Integration (Phase 3)
	s.processResources(ctx, startBoundary, end, hostName, osName)

	// 4. Value Integration (Phase 4: Business Value Ingestion)
	s.processValueUnits(ctx, startBoundary, end, hostName, osName)

	// 5. Fetch Tailscale State (Logs/Metrics only, no DB)
	s.collectTailscale(ctx)

	telemetry.Info("batch_complete")
}

func (s *Service) processResources(ctx context.Context, start, end time.Time, hostName, osName string) {
	if s.Resources == nil {
		return
	}

	costFactor, _ := s.Resources.GetCostFactor(ctx)
	carbonIntensity, _ := s.Resources.GetCarbonIntensity(ctx)

	// 1. Process Total Node Energy (Aggregate)
	nodeJoules, err := s.Resources.GetEnergyJoules(ctx, start, end)
	if err == nil && nodeJoules > 0 {
		s.recordMetricsForFeature(ctx, end, "node-total", nodeJoules, costFactor, carbonIntensity, hostName, osName)
	}

	// 2. Process K3s Pod Energy (Granular from Kepler)
	podMetrics, err := s.Resources.GetContainerEnergy(ctx, start, end)
	totalPodJoules := 0.0
	if err == nil {
		for container, joules := range podMetrics {
			featureID := mapContainerToFeature(container)
			if joules > 0 {
				totalPodJoules += joules
				s.recordMetricsForFeature(ctx, end, featureID, joules, costFactor, carbonIntensity, hostName, osName)
			}
		}
	}

	// 3. Process Host Attribution (For non-pod services like Ingestion, Proxy, MCPs)
	// We calculate the 'energy gap' (Total Node - K3s Pods) and attribute it to host services by CPU share.
	hostJoulesGap := nodeJoules - totalPodJoules
	if hostJoulesGap > 0 {
		hostServiceCPU, err := s.Resources.GetHostServiceCPU(ctx, start, end)
		if err == nil {
			totalHostCPU := 0.0
			for _, cpu := range hostServiceCPU {
				totalHostCPU += cpu
			}

			if totalHostCPU > 0 {
				for service, cpu := range hostServiceCPU {
					share := cpu / totalHostCPU
					attributedJoules := hostJoulesGap * share
					featureID := mapContainerToFeature(service) // uses same mapping
					s.recordMetricsForFeature(ctx, end, featureID, attributedJoules, costFactor, carbonIntensity, hostName, osName)
				}
			}
		}
	}
}

func (s *Service) processValueUnits(ctx context.Context, start, end time.Time, hostName, osName string) {
	if s.Resources == nil {
		return
	}

	valueUnits, err := s.Resources.GetValueUnits(ctx, start, end)
	if err != nil {
		telemetry.Error("value_units_fetch_failed", "error", err)
		return
	}

	for feature, count := range valueUnits {
		if count > 0 {
			metadata := map[string]interface{}{
				"host": hostName,
				"os":   osName,
			}
			_ = s.Store.RecordAnalyticsMetric(ctx, end, feature, KindValueUnit, count, "count", metadata)
			telemetry.Info("value_unit_recorded", "feature_id", feature, "count", count)
		}
	}
}

func (s *Service) recordMetricsForFeature(ctx context.Context, t time.Time, featureID string, joules, costFactor, carbonIntensity float64, hostName, osName string) {
	metadata := map[string]interface{}{
		"host": hostName,
		"os":   osName,
	}

	// Record Energy
	_ = s.Store.RecordAnalyticsMetric(ctx, t, featureID, KindEnergy, joules, "joules", metadata)

	// Record Cost
	cad := joules * costFactor
	_ = s.Store.RecordAnalyticsMetric(ctx, t, featureID, KindCost, cad, "cad", metadata)

	// Record Carbon
	// gCO2 = (Joules / 3,600,000) * gCO2/kWh
	gCO2 := (joules / 3600000.0) * carbonIntensity
	_ = s.Store.RecordAnalyticsMetric(ctx, t, featureID, KindCarbon, gCO2, "g_co2", metadata)

	telemetry.Info("feature_analytics_recorded", "feature_id", featureID, "joules", joules, "host", hostName)
}

func mapContainerToFeature(container string) string {
	// Simple mapping based on known service names (containers or systemd units)
	mapping := map[string]string{
		"ingestion":               "ingestion",
		"ingestion.service":       "ingestion",
		"proxy":                   "proxy",
		"proxy.service":           "proxy",
		"analytics":               "analytics-engine",
		"mcp-telemetry":           "agentic-telemetry",
		"mcp-pods":                "agentic-kubernetes",
		"mcp-hub":                 "agentic-hub",
		"postgresql":              "database-core",
		"postgres":                "database-core",
		"prometheus-server":       "observability-infra",
		"node-exporter":           "observability-infra",
		"query":                   "observability-infra",
		"compactor":               "observability-infra",
		"storegateway":            "observability-infra",
		"thanos-sidecar-manual":   "observability-infra",
		"grafana":                 "observability-ui",
		"loki":                    "observability-logs",
		"nginx":                   "observability-logs", // loki-gateway
		"tempo":                   "observability-traces",
		"opentelemetry-collector": "observability-otel",
		"openbao":                 "security-secrets",
		"openbao.service":         "security-secrets",
		"tailscale-gate":          "network-funnel",
	}

	if feature, ok := mapping[container]; ok {
		return feature
	}
	return container // fallback to raw container name
}

func getHostMetadata() (string, string, error) {
	// 1. Get Hostname from mounted host file
	hostnameBytes, err := os.ReadFile(pathHostname)
	if err != nil {
		return "", "", fmt.Errorf("failed to read %s: %w", pathHostname, err)
	}
	hostname := strings.TrimSpace(string(hostnameBytes))

	// 2. Get OS Name from mounted host os-release
	osReleaseBytes, err := os.ReadFile(pathOSRelease)
	if err != nil {
		return "", "", fmt.Errorf("failed to read %s: %w", pathOSRelease, err)
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

func (s *Service) collectAndStoreHostMetrics(ctx context.Context, start, end time.Time, hostName, osName string) {
	// 1. Fetch Samples
	cpuSamples, _ := s.Thanos.QueryRange(ctx, QueryCPUUtil, start, end, Step)
	tempSamples, _ := s.Thanos.QueryRange(ctx, QueryTemp, start, end, Step)
	ramSamples, _ := s.Thanos.QueryRange(ctx, QueryRAMUtil, start, end, Step)
	diskSamples, _ := s.Thanos.QueryRange(ctx, QueryDiskUsed, start, end, Step)
	netRXSamples, _ := s.Thanos.QueryRange(ctx, QueryNetRX, start, end, Step)
	netTXSamples, _ := s.Thanos.QueryRange(ctx, QueryNetTX, start, end, Step)

	// 2. Merge CPU and Temp (Robustly by truncating to minute)
	type timeKey int64
	cpuData := make(map[timeKey]map[string]interface{})

	for _, sa := range cpuSamples {
		key := timeKey(sa.Timestamp.Truncate(time.Minute).Unix())
		val, _ := strconv.ParseFloat(fmt.Sprintf("%v", sa.Payload["value"]), 64)
		cpuData[key] = map[string]interface{}{
			"usage":      val,
			"core_temps": make(map[string]float64),
		}
	}

	for _, sa := range tempSamples {
		key := timeKey(sa.Timestamp.Truncate(time.Minute).Unix())
		if data, ok := cpuData[key]; ok {
			labels := sa.Payload["labels"].(map[string]string)
			val, _ := strconv.ParseFloat(fmt.Sprintf("%v", sa.Payload["value"]), 64)
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
		s.Store.RecordMetric(ctx, time.Unix(int64(key), 0), hostName, osName, "cpu", payload)
	}

	// 4. Flush RAM
	for _, sa := range ramSamples {
		val, _ := strconv.ParseFloat(fmt.Sprintf("%v", sa.Payload["value"]), 64)
		s.Store.RecordMetric(ctx, sa.Timestamp, hostName, osName, "memory", map[string]interface{}{"used_percent": val})
	}

	// 5. Flush Disk (Stored as {"/": {"used_percent": ...}} for Grafana)
	for _, sa := range diskSamples {
		val, _ := strconv.ParseFloat(fmt.Sprintf("%v", sa.Payload["value"]), 64)
		payload := map[string]interface{}{
			"/": map[string]interface{}{"used_percent": val},
		}
		s.Store.RecordMetric(ctx, sa.Timestamp, hostName, osName, "disk", payload)
	}

	// 6. Merge and Flush Network (Stored as {"enp5s0": {"rx_bytes": ..., "tx_bytes": ...}} for Grafana)
	netData := make(map[timeKey]map[string]interface{})
	for _, sa := range netRXSamples {
		key := timeKey(sa.Timestamp.Truncate(time.Minute).Unix())
		val, _ := strconv.ParseInt(fmt.Sprintf("%v", sa.Payload["value"]), 10, 64)
		netData[key] = map[string]interface{}{
			"enp5s0": map[string]interface{}{"rx_bytes": val},
		}
	}
	for _, sa := range netTXSamples {
		key := timeKey(sa.Timestamp.Truncate(time.Minute).Unix())
		val, _ := strconv.ParseInt(fmt.Sprintf("%v", sa.Payload["value"]), 10, 64)
		if n, ok := netData[key]; ok {
			n["enp5s0"].(map[string]interface{})["tx_bytes"] = val
		} else {
			netData[key] = map[string]interface{}{
				"enp5s0": map[string]interface{}{"tx_bytes": val},
			}
		}
	}
	for key, payload := range netData {
		s.Store.RecordMetric(ctx, time.Unix(int64(key), 0), hostName, osName, "network", payload)
	}
}

func (s *Service) collectTailscale(ctx context.Context) {
	// 1. Funnel Status
	funnel, err := GetFunnelStatus(ctx)
	if err != nil {
		telemetry.Error("funnel_status_failed", "error", err)
		telemetry.AddInt64Counter(ctx, collErrors, 1)
	} else {
		telemetry.Info("tailscale_funnel_status", "active", funnel.Active, "target", funnel.Target)

		tailscaleMu.Lock()
		if funnel.Active {
			tailscaleActive = 1
		} else {
			tailscaleActive = 0
		}
		tailscaleMu.Unlock()
	}

	// 2. Node Status
	status, err := GetTailscaleStatus(ctx)
	if err != nil {
		telemetry.Error("tailscale_status_failed", "error", err)
		telemetry.AddInt64Counter(ctx, collErrors, 1)
	} else {
		telemetry.Info("tailscale_node_status", "backend_state", status["BackendState"])
	}
}
